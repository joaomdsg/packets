# CURR-ROADMAP — shortest path to the end-to-end review flow

> **Goal flow.** Start a session on a GitHub repo → coauthor a work-order in
> realtime with a real Claude Code harness in a producer → watch it get filled
> → inspect the work → leave adjustments → watch the adjustments get addressed
> → see tests pass → open a PR.
>
> This doc is the *near-term* roadmap that turns the built spine into that
> demoable flow. The full design is `DESIGN.md`; the round-by-round build log
> is `DESIGN.md` §0.2. Risk register: `RISKS.md`. Vision/economy: `VISION.md`.

## Where we are (grounded against the wired surface)

The goal flow maps onto two loops in the design. Only one is built today:

- **The catch-economy loop** (`DESIGN.md` §0.2, `internal/harness` + `internal/ledger`)
  — live order → real `claude` harness fills it → oracle catch → answer-with-a-test
  to kill a surviving mutant. **Built and wired end-to-end.**
- **The review-thread loop** (`DESIGN.md` §4, §11 P2/P3, §12.3, §14, §28) —
  inline comment on a line → fed to the harness as a turn → reply + fixup
  revision → resolve/outdated. **This is the flow the goal describes, and it is
  largely unbuilt.**

The build deliberately front-loaded the novel integrity spine (`DESIGN.md` §10:
"the build order de-risks the pipe and the integrity, not the review UX"). The
review-thread UX was sequenced last.

### Step-by-step status

| Step | Status | Evidence |
|------|--------|----------|
| (a) start session on a **gh repo** | PARTIAL | `-repo <local-path>` only, no URL→clone (`cmd/packets/main.go`); runtime "create session" inherits config (`internal/app/board.go`) |
| (b) coauthor work-order realtime w/ real harness | **WIRED** | Monaco authoring (`internal/app/authoring.go`) → `PlaceOrder` → `runLiveOrder` → `harness.RunProcess`/`RunContainer` (`internal/app/live.go`) |
| (c) see it filled | **WIRED** | live activity beats over SSE to the card (`internal/app/live.go`, `formatActivity`) |
| (d) inspect work | **WIRED** | Monaco base→fix diff island + cached verdict (`internal/app/review_surface.go`, `internal/app/live.go`) |
| (e) do adjustments | PARTIAL | only "submit a test to kill a surviving mutant" (`AnswerQuestion`, `internal/app/review_surface.go`); no "tell the agent what to change" |
| (f) see adjustments addressed | PARTIAL | the test re-runs and the question vanishes, but **the agent never re-edits** — no harness round-trip |
| (g) see tests pass | **WIRED** | catch cycle runs `go test ./...`, verdict resolves on the card (`internal/app/live.go`, `pipe.RunCatchCycle`) |
| (h) open a PR | ABSENT | no `gh pr`/push/land-action wiring; "Land" is a diagnostic verdict only |

**Net:** (b)(c)(d)(g) real; (a)(e)(f) partial-or-wrong-shape; (h) absent.

## The plan — three additive slices

Do **not** build the full `DESIGN.md` §14 thread/message projection or the §28
re-anchor machinery first. Reuse the live-harness pipe that already exists
(`runLiveOrder`) and add three thin slices. TDD per `CLAUDE.md` / `CONVENTIONS.md`
(`tdd-rygba`).

### Slice 1 — Comment→harness round-trip (keystone, the real work)

The one piece that makes it feel like reviewing a teammate. Converts (e) and (f)
from "submit a test" to "tell the agent what to fix, watch it fix."

- Add an anchored-comment entry point: `{file, line, text}` composes the
  `DESIGN.md` §12.3 turn template and dispatches it to the **same**
  `runLiveOrder` against the existing session HEAD, settling a new revision.
- Render adjustments as flat session-attached comments first. Full
  thread/outdated/re-anchor state (`DESIGN.md` §12.4, §28; schema §14) is
  deferred polish — **not** a prerequisite.
- This is an *additive reuse* of `runLiveOrder` (`internal/app/live.go`), not new
  architecture. It is the only architecturally non-trivial piece.

Refs: `DESIGN.md` §4 (PR⇄harness mapping), §12.3 (routing a comment back),
§11 P2.

### Slice 2 — `gh pr create` on approve (small, ~1–2 days)

Closes (h). Reuses the agntpr `x-access-token` push pattern.

- `landMode=pr` action: guard (open threads / red checks, overridable) → squash
  session revisions → push branch with a short-lived token → `gh pr create`.
- Mechanical; no new architecture.

Refs: `DESIGN.md` §16 (approve & land), §9.2 (landMode), §29.2 (merge queue /
Landed non-terminal — can be deferred; v1 can push direct branch + PR).

### Slice 3 — Repo-from-URL on session create (small, ~1–2 days)

Closes (a).

- Extend `-repo` (and the board "create session" path) to accept a URL: clone
  on create, checkout a fresh branch off `base_ref`.

Refs: `DESIGN.md` §15.2 (lifecycle: starting → clone + checkout branch),
`cmd/packets/main.go`, `internal/app/board.go`.

## Sequencing & estimate

1. **Slice 1** — substantial (the keystone). Build first; everything else is
   leverage on it.
2. **Slices 2 & 3** — small, independent, ~1–2 days each; parallelizable.

**Bottom line:** ~one substantial slice + two small ones from a coherent
end-to-end demo of the goal flow — *provided* adjustments render as
comment→harness turns rather than full GitHub-grade outdated-thread machinery.

## Explicitly deferred (not on this roadmap)

- Full thread/message relational projection + outdated/re-anchor (`DESIGN.md`
  §12.4, §14, §28).
- Merge-queue delivery + Landed→Merged lifecycle (`DESIGN.md` §29.2).
- Cross-process external-producer claims for *live* orders (today the claim path
  serves pre-baked backlog targets only; `DESIGN.md` §0.2).
- The rest of the trust economy: catch-weight, risk tiers, trust half-life,
  earned concurrency, Delegation Tiers (`DESIGN.md` §0.2, `VISION.md`).
