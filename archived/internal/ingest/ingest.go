// Package ingest is the trusted host-side intake of an untrusted peer's git
// objects: it validates a peer-supplied git bundle and unbundles it into a
// per-peer ref namespace in the host object store, so a later cage
// verification can resolve the claim's revisions WITHOUT the host ever fetching
// from a peer-controlled URL (no egress, no SSRF — council round 38).
//
// It is fail-closed and namespace-confined: every ingested ref is forced under
// refs/producers/<peerID>/* (legacy wire name — the on-disk namespace predates
// this rename and stays unchanged), so no peer-chosen ref name can touch the
// host's own refs or another peer's. A malformed, oversized, or
// wrongly-named submission is rejected and writes nothing.
package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ErrBundleInvalid marks a peer payload that is not a valid, unbundleable git
// bundle (garbage bytes, a truncated/corrupt pack, or missing prerequisites).
var ErrBundleInvalid = errors.New("ingest: peer bundle is invalid")

// ErrCapExceeded marks a peer bundle past the byte cap — rejected before any
// git work, so a pack/bundle bomb costs the host nothing.
var ErrCapExceeded = errors.New("ingest: peer bundle exceeds the size cap")

// ErrBadPeerID marks a peer id that is not a safe single ref segment —
// it could otherwise escape its namespace via the refspec (path traversal or
// extra ref segments) and write the host's or another peer's refs.
var ErrBadPeerID = errors.New("ingest: peer id is not a safe ref segment")

// safePeerID is the trusted ref-segment alphabet: it cannot contain '/', '.'
// runs that traverse, whitespace, or git refspec metacharacters, so it can only
// ever name one segment under refs/producers/.
var safePeerID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// IngestPeerObjects validates a peer's git bundle and unbundles it into
// refs/producers/<peerID>/* in the host store (an existing git repo). The
// order is cheapest-guard-first: id validation, then the byte cap (before any
// git work), then bundle verification, then a forced-refspec fetch that confines
// every ref to the peer namespace. Any failure writes nothing to the store.
func IngestPeerObjects(ctx context.Context, store, peerID string, bundle []byte, maxBytes int64) error {
	// A ".." anywhere is a refspec range/traversal metacharacter (even when the
	// alphabet check passes, e.g. "a..b"); a lone "." is the current-dir segment.
	// Reject both up front so the failure is an honest bad-id, not a downstream
	// "invalid bundle" once git refuses the refspec.
	if !safePeerSegment(peerID) {
		return fmt.Errorf("ingest: peer id %q: %w", peerID, ErrBadPeerID)
	}
	if maxBytes > 0 && int64(len(bundle)) > maxBytes {
		return fmt.Errorf("ingest: bundle is %d bytes (cap %d): %w", len(bundle), maxBytes, ErrCapExceeded)
	}

	tmp, err := os.CreateTemp("", "packets-ingest-*.bundle")
	if err != nil {
		return fmt.Errorf("ingest: temp bundle: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bundle); err != nil {
		tmp.Close()
		return fmt.Errorf("ingest: write bundle: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("ingest: close bundle: %w", err)
	}

	// Verify the bundle is well-formed before letting it touch the store; fold the
	// git stderr into the message text (one %w, on the sentinel).
	if out, err := git(ctx, "", "bundle", "verify", tmp.Name()); err != nil {
		return fmt.Errorf("ingest: bundle verify (%s): %w", strings.TrimSpace(out), ErrBundleInvalid)
	}

	// Force every bundle ref refs/X to refs/producers/<id>/X: a bundle naming
	// refs/heads/main lands at refs/producers/<id>/heads/main, never the host's
	// own refs/heads/main. --end-of-options guards the temp path; the refspec is
	// the namespacing. (refs/producers/ is the legacy wire name for this
	// namespace — the on-disk layout predates the peer rename.)
	refspec := "refs/*:refs/producers/" + peerID + "/*"
	if out, err := git(ctx, store, "fetch", "--no-tags", "--no-write-fetch-head",
		"--end-of-options", tmp.Name(), refspec); err != nil {
		return fmt.Errorf("ingest: unbundle into namespace (%s): %w", strings.TrimSpace(out), ErrBundleInvalid)
	}
	return nil
}

// safePeerSegment reports whether id is usable as a single ref segment under
// refs/producers/: the trusted alphabet, never a lone "." nor any ".." (a refspec
// range/traversal metacharacter), so it can only ever name one namespace segment.
func safePeerSegment(id string) bool {
	return safePeerID.MatchString(id) && id != "." && !strings.Contains(id, "..")
}

// PrunePeerObjects reclaims a peer's ingested objects once they are no
// longer needed, WITHOUT ever orphaning a pending claim. The economy-safe
// retention rule: a peer's ingested objects back its claims'
// revisions, so they MUST survive while any of that peer's claims is in
// flight; once none is (every claim resolved, or it only uploaded and never
// claimed), the whole namespace is dead weight. hasInFlightClaims is the caller's
// answer (e.g. ledger.ClaimsInFlight()>0 for the session) — when true this is a
// no-op, when false it deletes every ref under refs/producers/<peerID>/,
// making those objects unreachable and eligible for git's gc. It deletes only
// that one namespace, never the host's own refs nor another peer's.
//
// This is SESSION-granularity (prune-all-iff-none-in-flight), a strictly-safe
// simplification of the per-target rule: it never prunes while a claim pends, so
// it cannot orphan an in-flight target. Finer per-target pruning, and the actual
// disk reclamation via `git gc --prune`, are deferred follow-ups.
func PrunePeerObjects(ctx context.Context, store, peerID string, hasInFlightClaims bool) (deleted int, err error) {
	if !safePeerSegment(peerID) {
		return 0, fmt.Errorf("ingest: peer id %q: %w", peerID, ErrBadPeerID)
	}
	if hasInFlightClaims {
		return 0, nil // a pending claim's objects must survive — never orphan a verify
	}
	prefix := "refs/producers/" + peerID + "/"
	out, err := git(ctx, store, "for-each-ref", "--format=%(refname)", "--end-of-options", prefix)
	if err != nil {
		return 0, fmt.Errorf("ingest: list peer refs (%s): %w", strings.TrimSpace(out), err)
	}
	for _, ref := range strings.Fields(out) {
		if delOut, derr := git(ctx, store, "update-ref", "-d", "--end-of-options", ref); derr != nil {
			return deleted, fmt.Errorf("ingest: delete %s (%s): %w", ref, strings.TrimSpace(delOut), derr)
		}
		deleted++
	}
	return deleted, nil
}

// git runs a git command (in dir, or the process cwd when dir is empty) and
// returns its combined output plus an error carrying that output on failure.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
	}
	return out.String(), nil
}
