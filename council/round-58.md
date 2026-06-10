# Round 58 — the LAND/MERGE FLEET SURFACE thread (Landed ≠ Merged) — BUILDING — 2026-06-10

Trigger: the prep-bench thread (R57) shipped feature-complete (render + choose-to-
fund; reorder declared marginal since the Lead can already pick any item). R57's
council had QUEUED this thread as the next one — the CI/CD member's pick.

## The thread (council-queued in R57)

CI/CD's standing concern: "Landed ≠ done; only Merged through real CI is." The catch
cycle already computes an integration verdict per fix — pipe.LandState
{clean/conflict/checks_red} via integrateOnTip — and the live CARD shows it
(surface.RenderLand). But a Lead scanning the fleet board had NO way to see which
sessions are blocked from merging. This thread surfaces integration readiness across
the fleet.

Reachability (scouted): Land lived only in the per-tab c.Land cell + the cycle's
res.Land — not on liveEntry or the ledger, so the board couldn't read it. Slice 1
caches it (mirroring the R56 findings cache).

## Slice 1 (built, commit 2e6de9b)

liveEntry gained a `land` field (cached by the connect cycle via e.setLand, guarded
by the existing findingsMu — ephemeral, off the economy ledger, like the findings
cache). BoardRows reads it into CardRow.Land. BoardCard.View renders a per-session
`board-row__land` span ONLY when the verdict BLOCKS a merge — boardLand maps
conflict→("land-conflict","merge blocked: rebase") and checks_red→("land-checks-red",
"merge blocked: checks red"); clean/pending → nothing (the board stays calm, a
blocked session stands out — same gated idiom as the open-questions count). The
data-state hooks reuse R45's honest palette (conflict→amber, checks-red→muted loss).

Tests (board_land_internal_test.go): a conflict session + a checks-red session both
surface their blocked land span with the right state hook + "merge blocked"; a clean/
pending session surfaces none. Verified via the tests + full-repo -race gate (a
cache + gated board span mirroring the already-audited R56 findings-cache and
R48/R50/R55 board-span patterns — proportionate to size, no separate Blue/Audit
subagent).

This makes "which sessions can't land" visible across the fleet. Possible further
slice: a positive mergeable indicator or a fleet merge-readiness summary — judge
value vs declaring the thread complete (the blocked-states surface is the core gap).

## New clashes opened / resolved

None. The thread realizes the CI/CD member's early bold swing (integration
visibility) on existing plumbing, off the economy, server-render-testable.
