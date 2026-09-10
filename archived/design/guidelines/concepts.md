# Packets — the model & concepts

The conceptual model behind every surface in this design system. Vocabulary is binding: designs and code should use these words exactly.

## The thesis
Networking automated **forwarding** — line rate, no human per packet — but never automated away **inspection**: Wireshark, tcpdump, Snort exist because operators look constantly, out of band, on their own terms. Packets does the same for agent-written code changes. Automate forwarding; keep inspection first-class and sharp.

## The objects

- **Packet** — one agent-written change, from plain-words intent to delivery. Named (`rate-limiter`), revisioned (`rev2`), addressed. The product's atom; it has a whole replayable life (the timeline).
- **Addr** — the repo, in `owner/repo` form (`acme/edge-gateway`). A packet starts and lands at the same addr — the mark encodes this (ghost outline top-right → delivered fill bottom-right, same color, same edge).
- **Intent** — the human's prose goal. Unrunnable; the thing everything else must stay faithful to.
- **Handshake** — the runnable contract encoding intent, **authored independently of, and before, the code**. Conformance to it is automatic, at line rate. Strength is a gradient: examples → properties/contracts → proof — bought by blast radius.
- **Implementation tests** — the unit/integration/e2e the coding agent writes. **Evidence, never the contract.** "The agent's tests pass" ≠ "the handshake is met" (same-author tests are homework-grading). Authority ⊥ scope: a property unit test can be a handshake term; a flaky e2e can't.
- **Lane** — QoS class derived from the dependency graph (never self-reported): best-effort → standard → strict → irreversible. Blast radius buys two things: more gates, and a stronger required handshake. Irreversible is verified BEFORE prod (staged, canary, mandatory human) — never "verify in prod", because production is not reversible (data, money, sent emails).
- **Blast radius** — measured coupling; what decides holds. Never vibes.
- **Socket** — an addr's live attachment to the fabric, auto-managed for the addr's lifetime; carries the addr's packets. Keeps a warm base session (listening), parks to a resumable ticket when idle. The addr stays the durable identity independent of any socket; a socket both accepts (compose activity, peer claims) and originates (sends packets).
- **Packet / send** — draft (being composed) → packet (sealed with prompt, base rev, handshake as headers) → send over the socket. Every packet requires its own handshake, consumed by that send.
- **Peer** — a remote endpoint admitted by a Grant, confined to publishing claims into its own session subtree; never mints.
- **Source** — provenance of a mint (`"connect"` or `"wo:<id>"`).

## The lifecycle (= the state grammar = the mark)
compose (ghost outline) → forward/in-flight (bright cyan) → verify (green) → **held** (amber advisory / red blocking) → deliver (dark cyan fill). Delivered means **ACK'd healthy in prod** (`packets deployed` / `packets regressed`) — not merged.

## The gauntlet (standing inspection)
Six gates; each lane runs only what its radius warrants:
1. **Intent fidelity** — the lone human residual; does the handshake capture what was wanted?
2. **Handshake conformance** — automatic, line-rate ("does it work?" is `make test`, not a review).
3. **Handshake tightness** — mutate the code; does the handshake scream? (mutation vs SPEC).
4. **Build · types · lint · security** — deterministic, zero human cost.
5. **Test sensitivity** — mutate vs the agent's tests; are they hollow? (mutation vs TESTS).
6. **Independent check** — *method* diversity, never a second model: static analysis, property tests, contracts, human-on-sample. Two LLM reviewers share blind spots — failures correlate.

## Inspection has four modes
- **Pull** (human, on demand — ≈ Wireshark): crack open any packet, anytime; cheap enough to do out of curiosity. Looking never slows forwarding (every packet is mirrored — a SPAN tap).
- **Push** (system → you — ≈ alerting): held packets, budgeted interrupts.
- **Standing** (machine, always-on — ≈ IDS/DPI): the gauntlet + watches/capture triggers, each carrying a precision score; noisy triggers lose the right to interrupt.
- **Adversarial** (red-team — ≈ pen-test): actively tries to break packets; seeds probes to keep everyone honest.

## Attention economics
Interrupts are budgeted (n/10 per week); you make ~3 real decisions a day while ~200 packets ship. An empty queue is success, not idleness. Calibration draws (skim a random sample of auto-forwarded packets) keep trust measured, not assumed. The human's job moved: from reading every diff (the ceiling that rots into rubber-stamping) to authoring handshakes, judging intent-fidelity, setting lanes and triggers — and inspecting by pull to stay sharp.

## Honest residuals (the three dials)
Correlated failure can be reduced, never zeroed; a wrong-but-precise handshake still ships if no human reads the intent. The claim is never "autonomy is safe" — it's "safe up to your handshake's fidelity, the method-diversity of your checks, and the inspection read-rate you'll pay."

## The one test for any feature
A change should **forward** autonomously unless inspection must hold it. Maximise what forwards safely — push work *up* into the handshake or *down* into a standing check, keep agent tests as evidence only, and make pull-inspection cheap enough that operators drill in for fun.
