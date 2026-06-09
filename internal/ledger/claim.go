package ledger

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joaomdsg/packets/internal/fabric"
)

// subjectKindClaim is the claim-subtree token for a unit of work submitted for
// verification — distinct from the minted-catch kind, so a claim and a catch can
// never be confused on the bus.
const subjectKindClaim = "work"

// ClaimRecord is an untrusted producer's work-submission: the revs and anchored
// line (a Target) the host must VERIFY before it mints anything. It deliberately
// carries NO test command — the host fixes what it runs, so a producer cannot
// choose the command executed on its behalf — and it is published on the claim
// subtree, never the minted subtree, so a claim credits nothing until a host-run
// oracle confirms it.
type ClaimRecord struct {
	Target Target `json:"target"`
}

// PublishClaim emits a producer's work-submission on the claim subtree for
// session+instance and returns its stream sequence. It targets StatusClaim, not
// StatusMinted, so it never enters the economy projection — the host consumes it,
// verifies it, and only then mints through the authoritative catch path.
func PublishClaim(ctx context.Context, f *fabric.Fabric, session, instance string, c ClaimRecord) (uint64, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return 0, fmt.Errorf("ledger: encode claim: %v", err)
	}
	return f.Publish(ctx, fabric.EventSubject(session, instance, fabric.StatusClaim, subjectKindClaim), data)
}

// DecodeClaim decodes a producer work-submission payload from the bus.
func DecodeClaim(data []byte) (ClaimRecord, error) {
	var c ClaimRecord
	if err := json.Unmarshal(data, &c); err != nil {
		return ClaimRecord{}, fmt.Errorf("ledger: decode claim: %v", err)
	}
	return c, nil
}

// Verifier turns an untrusted claim into a verdict: a *CatchRecord to mint when
// the oracle confirms a real catch, or nil to mint nothing. It is the SEAM where
// the host's verifier plugs in — in production a sandboxed run of the mutation
// oracle (#6c, the only place untrusted code executes); the consumer here never
// runs that code itself.
type Verifier func(ClaimRecord) (*CatchRecord, error)

// ConsumeClaims subscribes to this log's claim subtree and, for each submitted
// claim, runs verify and mints any returned record through the authoritative
// Append path — so the catch-only gate and the identity dedup apply: a verdict of
// nil mints nothing, and a re-submitted claim (same verified identity) mints
// nothing more. A malformed claim or a verifier error is skipped so one bad claim
// can't stall the stream; a mint that the gate refuses (duplicate/non-catch) is
// also skipped — best-effort, matching the in-process mint. It blocks until ctx
// is canceled (the only teardown), then returns; the caller runs it in a
// goroutine and MUST cancel ctx when done.
//
// ackWait is the durable consumer's redelivery window: a claim is acked only
// after verify returns, so ackWait MUST exceed the verifier's per-claim deadline,
// or a slow verify is redelivered into a concurrent re-verify. The caller wires
// the two together (the cage verify deadline and this ackWait above it).
func (l *Log) ConsumeClaims(ctx context.Context, verify Verifier, ackWait time.Duration, adm *Admission) error {
	filter := fabric.EventSubject(l.session, l.instance, fabric.StatusClaim, ">")

	// One token bucket per producer (this log is one session+instance). A nil
	// Admission means no rate limit. The clock is the admission's (time.Now in prod).
	var bucket *tokenBucket
	var now func() time.Time
	if adm != nil {
		now = adm.clock()
		bucket = newTokenBucket(adm.Burst, adm.RatePerSec, now())
	}

	return l.f.ConsumeDurable(ctx, claimDurable(l.session, l.instance), filter, ackWait, func(e fabric.Event) error {
		claim, err := DecodeClaim(e.Data)
		if err != nil {
			return nil // a malformed claim is skipped (acked), not redelivered forever
		}
		// Skip the (expensive, sandboxed) verify if the target is already minted:
		// Append would refuse the re-mint anyway, but only AFTER burning a cage run.
		// On a read error, fall through to verify (do the work) — the gate still
		// dedupes the mint, so a stale read costs compute, never correctness.
		// This MUST precede the rate check, so a duplicate doesn't spend a token.
		if records, rerr := l.Records(); rerr == nil && targetAlreadyMinted(records, claim.Target) {
			return nil
		}
		// Per-producer rate limit: a flood beyond the burst is ack-dropped before
		// the verifier (the scarce compute), so the producer can't starve the host.
		if bucket != nil && !bucket.allow(now()) {
			return nil
		}
		// Global concurrency cap: bound the total concurrent verifies across all
		// producers. QUEUE (block) for a slot rather than reject — claims are
		// durable, so backpressure loses no work; release the slot when the handle
		// returns (after verify+Append). On ctx cancel, return an error so the
		// claim is not acked and redelivers later, never lost to a shutdown.
		if adm != nil && adm.Concurrency != nil {
			select {
			case adm.Concurrency <- struct{}{}:
				defer func() { <-adm.Concurrency }()
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		rec, err := verify(claim)
		if err != nil || rec == nil {
			return nil // verifier error or no-catch verdict: nothing to mint, ack and move on
		}
		_ = l.Append(*rec) // a gate-refused (duplicate/non-catch) mint is best-effort, matching the in-process path
		return nil
	})
}

// targetAlreadyMinted reports whether the committed economy already holds a catch
// for this claim's target, matched on the catch IDENTITY (BeforeRev, AfterRev,
// Path, Line) — the same fields Append dedupes on. TipRev is not part of the
// identity, so it is not compared.
func targetAlreadyMinted(records []CatchRecord, t Target) bool {
	for _, r := range records {
		if r.BeforeRev == t.BaseRev && r.AfterRev == t.FixRev && r.Path == t.Path && r.Line == t.Line {
			return true
		}
	}
	return false
}

// ClaimsInFlight counts the DISTINCT claim targets submitted on this log's claim
// subtree that are not yet minted — the producers' pending "bets". It is kept
// strictly separate from the confirmed economy (Balance/Records): a pending claim
// is never a confirmed catch (the two-scores invariant), and a target moves out
// of "in flight" the moment it mints. Duplicate replays of one target count once.
//
// A rejected claim has no terminal marker yet, so a rejected target lingers as
// in-flight until slice C3 adds explicit rejection — accepted C1 behavior.
func (l *Log) ClaimsInFlight() (int, error) {
	filter := fabric.EventSubject(l.session, l.instance, fabric.StatusClaim, ">")
	events, err := l.f.ReplaySubject(context.Background(), filter)
	if err != nil {
		return 0, err
	}
	records, err := l.Records()
	if err != nil {
		return 0, err
	}

	// Dedupe by the catch IDENTITY (BaseRev,FixRev,Path,Line) — the same tuple
	// targetAlreadyMinted and Append key on — so a replay (or a tip-only variation)
	// of one unit of work counts once, not twice.
	type identity struct {
		base, fix, path string
		line            int
	}
	seen := make(map[identity]bool)
	for _, e := range events {
		claim, derr := DecodeClaim(e.Data)
		if derr != nil {
			continue // a malformed claim event is not a claim in flight
		}
		if !targetAlreadyMinted(records, claim.Target) {
			t := claim.Target
			seen[identity{t.BaseRev, t.FixRev, t.Path, t.Line}] = true
		}
	}
	return len(seen), nil
}

// claimDurable is the stable durable-consumer name for a log's claim subtree,
// derived from its session+instance so a restart resumes the SAME consumer
// (resuming past already-processed claims). NATS durable names forbid the subject
// separators and wildcards, so any are mapped to '_'.
func claimDurable(session, instance string) string {
	return "claims_" + durableToken(session) + "_" + durableToken(instance)
}

func durableToken(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '.', '*', '>', ' ', '/', '\t', '\n':
			return '_'
		default:
			return r
		}
	}, s)
}
