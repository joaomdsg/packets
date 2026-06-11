# Round 84 — producer-auth gate: GC-by-resolved as a post-verdict hook — 2026-06-11

Trigger: with the producer-auth boundary in place (R81–R83), wire the bundle GC
the maintainer chose to run on a POST-VERDICT HOOK (prune the moment a claim
resolves), on top of the periodic sweep that already existed.

## What already existed

`app.PruneIdleProducers` + `StartProducerGC` (a periodic ticker) already reclaim
each idle producer's `refs/producers/<key>/*` gated on `ClaimsInFlight()==0`, with
the economy-safe retention + fail-toward-keep rules (council R39). The periodic
sweep was the only trigger; this round adds the prompt one.

## The change

- `ledger.Admission` gains `OnResolved func(session string)` — called AFTER a
  claim reaches a DURABLE verdict (mint via `Append`, or rejection via a
  no-catch/unverifiable `PublishClaimVerdict`), so the GC sees the up-to-date
  in-flight count. A TRANSIENT error (cage flake/timeout) leaves the claim in
  flight and deliberately does NOT fire — pruning then would be gated to a no-op
  anyway, but firing only on true resolution keeps the semantics honest.
- `ConsumeClaims` calls `resolved()` at exactly the three resolution exits
  (unverifiable-reject, no-catch-reject, mint), never the transient path.
- `app/gc.go` factors the per-session prune into `pruneProducerIfIdle` (shared by
  the sweep) + `pruneProducerSession(ctx, key)` (the hook's key-only entry point);
  `PruneIdleProducers` now calls the shared unit.
- `StartCageClaimConsumers` sets `adm.OnResolved = pruneProducerSession(ctx, …)`,
  so the instant a producer's last claim resolves, its objects are reclaimed —
  the periodic sweep becomes a backstop for the upload-but-never-claim leftovers,
  not the primary reclamation path.

RED test: `TestClaimResolution_reclaimsTheProducersObjectsViaThePostVerdictHook` —
through the PRODUCTION wiring (`StartCageClaimConsumers` + the `blessingRunner`
cage stub), an ingested producer bundle's `refs/producers/default/*` is pruned
after the claim mints, with NO periodic sweep started and NO manual
`PruneIdleProducers` call. Without the OnResolved wiring the ref would survive, so
the test genuinely exercises the hook.

## Scope held

The TOCTOU / fail-toward-keep / economy-safety analysis is unchanged — the hook
reuses the same `pruneProducerIfIdle` unit and its `ClaimsInFlight`-gated,
read-error-skipping, empty-repo-skipping rules (R39). The hook is purely an
earlier TRIGGER, not a new policy. Per-producer flood-defenses (bundle rate-limit
+ aggregate quota) are R85.

## Verdict

`internal/ledger` + `internal/app` green with `-race`. A resolved claim now
reclaims its producer's objects immediately; the periodic sweep remains the
backstop. Next: R85 — the bundle flood-defenses the auth boundary makes
attributable, completing the gate's RISKS.md fix sequence.

## New clashes opened / resolved

Resolved: GC-by-resolved fires on the post-verdict hook (R78→R84), reusing the
existing economy-safe prune unit; only the trigger is new. The periodic
`StartProducerGC` is retained for upload-without-claim leftovers.
</content>
