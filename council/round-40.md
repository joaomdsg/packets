# Round 40 — direction after #6c feature-complete; the live boundary is maintainer-gated — CONVERGED — 2026-06-10

Trigger: #6c (the untrusted-claim verification boundary) is feature-complete and
its housekeeping (GC-by-resolved) is wired live. The ~15-tick #6 thread has
reached its natural end. This round picks the next direction.

HARD CONSTRAINT (the load-bearing fact): authorizing the LIVE cross-process
boundary — standing up a network listener that accepts untrusted producer
connections, and wiring the #6a producer auth (ProducerGrant/StartListening,
which is BUILT in the fabric but not wired to the live HTTP server) into the
served surface — is HARD-GATED on explicit MAINTAINER SIGN-OFF (council/06-plan.md,
rounds 28 & 32, reaffirmed). The autonomous loop must NOT cross it. So the next
slice must be NON-GATED.

Panelists: Systems/Economy, Pragmatic TDD, Product/Vision.

## Per panelist

- ⚙️ Systems: finish the GC story — per-target pruning + actual `git gc --prune`
  disk reclaim + a TTL reap for abandoned bundles. Judged the perf hatch
  premature (single-viewer) and VISION pivots near the gate.
- 🧪 Pragmatic TDD: GC-disk-reclaim is BRITTLE — `git gc` is nondeterministic
  (compression/reachability/mode), exactly why R39 deferred byte-measured bounds;
  a "reclaim N bytes" test is flaky. The perf hatch (debounce/incremental fold)
  is timing/race-flaky offline. Clashes D/E/H/I are post-build, not offline-
  testable. THE clean win: the NON-ASCII re-anchor/diff correctness gap
  (RISKS.md) — a PURE, deterministic, offline fix (`-c core.quotepath=false` in
  reanchor's name-status + diff's diff) closing a real silent mis-handling of
  non-ASCII paths (phantom "Same" instead of Moved/Outdated). Load-bearing test:
  a real `café.txt` rename driven end-to-end must follow the rename, not phantom-
  resolve.
- 🎨 Product/Vision: build the human-facing BOARD / management-sim experience
  (render the fleet board from the stream, a real work-order round-trip, treasury
  + leverage visible, the claim lifecycle cluster, keyboard nav) — highest
  product payoff, makes the deep plumbing legible to a Lead, and settles the open
  product clashes D (fleet 1:N scale), E (Prep Bench), H (Trust Ledger framing)
  which can only be judged on a playable surface. Non-gated. But a LARGE,
  multi-part UX slice.

## Chair adjudication — CONVERGED

1. The LIVE boundary stays maintainer-gated — not proposed (constraint honored).
2. GC-disk-reclaim is OUT as the next slice: TDD's brittleness argument is
   decisive (git-gc nondeterminism), and the session-granularity GC already
   reclaims the refs (the economy-safe step); actual `git gc` is deferrable ops
   (a cron/manual op), not a TDD-clean slice.
3. The BOARD is the right next MAJOR THREAD (highest strategic value, settles
   real product clashes) — but it is a large, multi-slice UX effort and the
   product's human face; it deserves its OWN scoping sequence and very likely a
   maintainer UX steer, not a blind autonomous start. Defer to its own thread.
4. IMMEDIATE NEXT SLICE = TDD's pick: the NON-ASCII re-anchor/diff correctness
   fix. It is the smallest, most deterministic, least-flaky, non-gated slice; it
   closes a real logged correctness gap (soundness on non-ASCII repos); and both
   packages share the defect so it lands coherently in one slice.

## Decision

- NEXT BUILD: pin `-c core.quotepath=false` on the git invocations in
  internal/reanchor (fileStatus name-status) AND internal/diff (diff.Compute), so
  a non-ASCII Anchor.Path matches git's output instead of falsely reading "Same".
  Load-bearing RED: a real non-ASCII-path rename driven through the re-anchor
  cycle resolves as Moved/Outdated (the honest verdict), not a phantom Same; +
  diff.Compute returns the real (unquoted) path. Clears the RISKS.md entry.
- AFTER: open the BOARD thread with its own round (scope the thinnest playable
  board slice on existing plumbing; flag the maintainer-UX-steer question).
- DEFERRED (unchanged): the live boundary (maintainer-gated), GC disk reclaim
  (ops/brittle), the C3b2b perf hatch (premature), per-target GC.

## New clashes opened / resolved

None. Recorded that the #6 thread has reached its maintainer-sign-off gate (the
loop advances only non-gated work from here) and that the Board is the next major
thread pending its own scoping round.
