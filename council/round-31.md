# Round 31 — #6 opens: the cross-process producer boundary — converged on sequencing, one new clash on the trust model — 2026-06-08

Trigger: maintainer convened the council on roadmap #6 (the cross-process
producer + security trio) to run deliberation rounds until convergence,
after #1–#5 shipped. Decision round — no new build evidence.

Panelists present: all six, re-summoned from the §1 seeds.

## Per panelist

- 🎨 UX (CONVERGE): listen+authz first is correctly invisible to UX (no
  approval surface until the trio lands). One non-blocking rider for the
  trio effort: model permission-broker approvals as fabric EVENTS
  (subscribable, causal — the broker waits on the approve event), never an
  opaque out-of-band RPC the SSE board can't render.
- 🎮 Game design (CONVERGE, with a binding rider): the cross-process fleet
  is the literal shop floor — but the boundary is exactly where a hit-rate
  gets faked. Keep MINT authority on the VERIFIER side: a producer emits
  CLAIMS, the economy mints only what mutation-verification confirms. If
  "authorize to subtree" is read as "producer may publish minted catches,"
  the economy quietly becomes self-report.
- ⚙️ Systems/economy (CONVERGE, authz not severable): authn+authz land in
  the SAME build (no "authenticate now, authorize later" open-mint window).
  Three degenerate strategies to close: subtree-jumping (per-producer authz
  pinned to its own session.<key>.events.<instance>.> — the load-bearing
  control), replay/double-mint (pin instance, idempotency on
  (subject,event-id)), and producer-class forgery (the catch's class must
  derive from the AUTHORIZED subject, never producer payload).
- 🧪 Pragmatic TDD (CONVERGE): the two halves fall on opposite sides of the
  testability line and the sequencing respects it. RED-testable now (real
  loopback listener): forged/unauthenticated-producer rejection,
  subtree-confinement (A credentialed for A rejected publishing to B),
  authorized happy-path (reuse the #2 equivalence lock as oracle), negative
  subject-shapes. NOT unit-testable: seccomp/netns/broker (integration:
  attempt-the-forbidden-syscall → assert EPERM; never assert a config
  flag). Carve SECRET-SCRUB OUT of the trio gate — it IS testable (seed a
  secret across full history, scrub, assert gone AND projection still
  equivalent).
- 🚀 CI/CD & delivery (CONVERGE, with a split): producer authn/authz
  against the IN-PROCESS fabric is the safe next build; flipping the
  listener to actually BIND a reachable socket for an untrusted producer
  stays hard-gated on the trio — a half-built door reads as finished and is
  worse than none. Trio order: netns + host egress proxy (default-deny,
  cheapest blast-radius cap) → seccomp/LSM → broker (highest-trust, last;
  inherits §15/§19 scrutiny).
- 🔧 Refactoring (CONVERGE): listen-mode must be ADDITIVE — a new opt on
  fabric.Start; the in-process DontListen default stays byte-identical and
  is the regression oracle (the #1–#5 suite + the state-equivalence lock
  green is the gate-to-merge). Do NOT flip the default to listen to
  dogfood. Authz is a behavior-preserving NARROWING on the new path only
  (in-process keeps full reach; listeners get a strict subtree subset).

## Clashes touched

- OS-process-boundary sequencing — RESOLVED 6/6: build producer authn AND
  per-subtree authz together as in-process-proven, TDD-able machinery
  (default DontListen unchanged, listener tested on loopback only); GATE
  exposing a bound socket to an untrusted producer on the trio +
  full-history scrub. Round 28's hard-gate holds.
- secret-scrub placement — RESOLVED: carved OUT of the trio's
  "manual/integration" bucket into its own TDD-able slice (only the kernel
  members of the trio are genuinely non-unit-testable).

## Verdicts updated

None flip. The thesis stays PROVEN; #6 is leverage on the proven economy,
still hard-gated at the OS boundary.

## New clashes opened

- **CROSS-BOUNDARY TRUST MODEL (target-level, UNRESOLVED).** What does a
  cross-process producer actually publish? Game design wants producers to
  emit only unverified CLAIMS, with the host running the mutation oracle
  and minting (mint stays verifier-side). Systems' framing (confine the
  producer to its subtree, derive class from the authorized subject) reads
  as producers publishing CONFINED catch events directly. These differ on
  where verification authority sits relative to the boundary, and it is the
  load-bearing economy-integrity question. Not settled this round — the
  next round must adjudicate claim-submission vs confined-mint before the
  authz schema (what subjects/event-kinds a producer may publish) can be
  specified.

## Decisions

1. CONVERGED PATH (sequencing, 6/6): [#6a NEXT, once the trust model is
   settled] fabric listen-mode as an ADDITIVE opt + producer authn + per-
   subtree authz, landed together behind a flag, in-process default
   unchanged and proven by the equivalence lock; RED tests = forged-
   rejection, subtree-confinement, authorized-equivalence, negative
   subject-shapes. [#6b] full-history secret-scrub — its OWN TDD slice, not
   gated with the kernel trio. [#6c, HARD-GATED] binding a socket for an
   untrusted producer, behind the trio in order: egress-proxy → seccomp/LSM
   → out-of-container broker (approvals as fabric events per UX).
2. BLOCKER on #6a: the cross-boundary trust-model clash must resolve first
   — it determines the authz schema. Round 32's charge.
3. Carried riders (6/6): authz never severable from listen; mint
   idempotency on (subject,event-id); class-from-subject not payload;
   in-process default is the regression oracle; no unit test may claim to
   verify the kernel trio.

NOT fully converged: the path/sequencing is 6/6, but the trust-model clash
is open. Next event is ROUND 32 — adjudicate claim-submission vs
confined-mint, then the authz schema follows. No build until the council
converges AND the maintainer authorizes the gated boundary.
