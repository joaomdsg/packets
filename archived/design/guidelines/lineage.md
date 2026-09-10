# Packets — intellectual lineage

Dijkstra → Farley → Packets. Useful for marketing narrative, pitch material, and for understanding *why* the model is shaped this way. (A "lineage page" design is a tracked follow-up: scroll timeline, each era a packet state — outline → bright → dark.)

## Dijkstra (the axiom)
- "Testing shows the presence, never the absence of bugs" → why the handshake isn't just tests: mutation testing (falsify the test suite itself), properties over examples, guardrails.
- EWD667 (natural language can't carry precision) → Packets' bridge: plain words are the **interface**, the formal handshake is the **artifact**. The agent self-flag ("the handshake only gives examples") is a packet discovering that imprecision at runtime.
- Separation of concerns → split into *parties*: correctness is the loop's concern (mechanized falsification); judgment (blast radius, intent, taste) is the human's, at inspection.
- "We have small heads" → the attention budget, 50 years early. Blast-radius routing is small-heads engineering.
- THE-system layers → the inspector's layer-by-layer decode (intent / handshake / conformance / evidence / provenance).

## Farley (the process — Modern Software Engineering / Continuous Delivery)
- Pipeline = definitive statement of releasability → the handshake is releasability made explicit, per change.
- The pipeline as falsification mechanism; engineering as experiments → the forward loop (compose → verify → revise) is "optimize for learning" at machine speed.
- Change Approval Boards correlate *negatively* with quality; capture human decisions in the pipeline → HELD is a captured decision with an audit trail, not a review queue.
- Every commit creates a release candidate → the packet is the release candidate promoted to a durable, addressed noun.
- Small batches → the packet as unit of change.
- Deploy ≠ release ≠ value → delivered = ACK'd healthy in prod (`packets deployed` / `packets regressed`).
- "Branches tell lies" (only integrated code tells the truth) → 200 agent changes/day = 200 branches; packets re-verify against the integrated state; collisions are a first-class concept.
- Pair-programming navigator (his review replacement) → the agent self-flag is the navigator's voice, persisted as data; annotations are pairing made asynchronous.
- <1h definitive feedback → lane floors and loop cadence as enforced budgets.

## The synthesis
- Dijkstra 1970: **"prove it."** Farley 2021: **"pipeline it."** Packets 2026: **"let the machines prove it; spend your small head on judgment."**
- One-line lineage claim: CD made releasability an engineering property; **Packets makes attention one** — measurement, budgets, precision scores, and an audit trail for the one resource CD left unmanaged.
- Extension beyond Farley: his pipeline falsifies the code; Packets also falsifies the *checks* (mutation vs handshake, mutation vs tests) — Popper applied one level down.
