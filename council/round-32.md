# Round 32 — #6 trust model RESOLVED: claim-submission, host-side mint — the council CONVERGES on #6's design — 2026-06-08

Trigger: round 31 converged 6/6 on #6 sequencing but opened one target-level
clash — the cross-boundary trust model. This round adjudicates it. Decision
round, no build evidence.

Panelists polled (the load-bearing five): Systems/economy, Game design,
Pragmatic TDD, CI/CD, Refactoring. UX abstained (its round-31 broker-events
rider is untouched by this clash) and stands by its round-31 converge.

The clash: what does a cross-process producer publish? (a) CLAIM-SUBMISSION
— producer emits unverified claims, the trusted HOST runs the mutation
oracle and MINTS; vs (b) CONFINED-MINT — producer publishes catch events
directly, confined to its subtree.

## Per panelist — UNANIMOUS for (a)

- ⚙️ Systems (a, CONVERGE): authz-confinement bounds WHERE a producer
  writes, not WHETHER the oracle was beaten; (b) leaves the
  mint-without-verification hole wide open (publish a well-formed catch that
  never killed a mutant — the claimant controls the denominator). Under (a)
  a producer cannot write a CatchRecord at all, so the cheat is
  unrepresentable, not policed. Folds its round-31 framing INTO (a): keep
  subtree-confinement but apply it to CLAIM subjects; reserve mint
  (`...minted.catch`) subjects to the HOST identity only.
- 🎮 Game design (a, CONVERGE): a score the player writes about itself is a
  brag, not a score; (b) makes hit-rate self-attested. (a) is the same mint
  authority the in-process path already has, with the OS boundary drawn
  OUTSIDE it. Rider: confirmed-only hit-rate — claims-in-flight may show as
  gray "pending" (bets placed), but the hit-rate stat reads ONLY host-minted
  CatchRecords; never two scores on screen.
- 🧪 Pragmatic TDD (a, CONVERGE): the oracle exists precisely because the
  actor's self-report is untrusted; (b) re-introduces the test-theater the
  oracle defeats, with a subtree ACL stapled on. The host runs the oracle in
  ITS trust domain, not the producer's sandbox. Hard requirements: the
  host re-run must be DETERMINISTIC/reproducible (pin revs, mutant
  seed/ordering, the survivor-set the claim anchors) — non-reproducible →
  reject; and exactly ONE oracle implementation, invoked identically
  in-process and on-submission (a second verifier is a mint-laundering hole).
- 🚀 CI/CD (a, CONVERGE): this is the merge queue — (b) is letting a PR
  author flip their own check green and self-merge. Claim = open PR; host
  oracle = the trusted gate; mint = the merge commit (exists only after the
  gate runs). Verification fans out (stateless, parallel); only the
  mint/append serializes — reuse the existing single tip-append lane.
  Guardrail: verify a claim against the TIP at mint time (stale-claim
  re-validation), like re-running checks on the rebased tip.
- 🔧 Refactoring (a, CONVERGE): (a) leaves today's mint path
  (`oracle(work) → CatchRecord → ledger.Append`) literally unchanged — only
  the INPUT provenance moves (a cross-boundary claim instead of in-proc
  work), and input is upstream of the mint. (b) moves mint authority across
  the trust boundary and forks the minter into the least-trusted region — a
  rewrite wearing a refactor's clothes. Lock it with an EQUIVALENCE test:
  same oracle verdict → identical projection whether the work was generated
  in-proc or arrived as a claim.

## Clashes touched

- CROSS-BOUNDARY TRUST MODEL — RESOLVED 5/5 (UX abstain) toward (a)
  claim-submission. The unifying principle: ONLY THE VERIFIER MINTS, made
  structural (the producer has no capability to write a minted event), not
  policed. Round 31's confined-mint reading is COLLAPSED — confinement
  survives, but applied to claim subjects, not mint subjects.

## Verdicts updated

None flip. The founding premise (trust the mutation oracle, never the
actor's self-report) is REAFFIRMED and now extends across the OS boundary.

## New clashes opened

NONE at target level. The council is CONVERGED on #6's design. Residual
refinements (all compatible, folded into the plan): claim rate-limit/quota
so verification compute can't be exhausted; content-addressed claims (diff
hash) so the thing verified == the thing minted; confirmed-only hit-rate
with gray pending claims; verify-at-tip.

## Decisions — the converged #6 design

1. TRUST MODEL: claim-submission. A cross-process producer publishes only
   CLAIMS; the trusted host re-runs the single mutation-oracle
   implementation and mints. Only-the-verifier-mints, structurally enforced.
2. AUTHZ SCHEMA (derived): a producer authenticated for session <key>,
   instance <inst> may PUBLISH only to its own CLAIM subtree — a new
   `claim` status token, `packets.session.<key>.events.<inst>.claim.>` —
   and to nothing else. The authoritative `...minted.>` subjects are
   reserved to the HOST identity; producers have NO capability to publish
   them. This closes subtree-jumping AND mint-without-verification in one
   control.
3. The full converged build sequence is recorded in
   [`06-plan.md`](06-plan.md).
4. GATE UNCHANGED: this is the DESIGN. Building #6 remains hard-gated on
   the security trio AND explicit maintainer authorization. The council has
   produced the plan; it has not authorized the boundary.

CONVERGED (#6 design, this is the trace-forward the council was charged to
produce): claim-submission + host-side single-minter, with the round-31
sequencing (additive listen-mode, authn+authz together, in-process default
as the regression oracle, secret-scrub as its own slice, the trio
hard-gated below the container). Next event is NOT another round — it is the
maintainer's decision on whether/when to authorize the gated build.
