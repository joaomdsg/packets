# Round 87 — producer-auth: prove the boundary END TO END over a real socket — 2026-06-11

Trigger: the producer-auth gate (R81–R86) was tested by COMPOSITION — R81 proved
a grant credential can CONNECT, R82's lifecycle tests publish IN-PROCESS, and the
grant-confinement itself is fabric-tested. Nothing drove the full authenticated
cross-process path in one test. This round closes that assurance gap.

## The test

`TestProducer_authenticatedExternalClaimPublishMintsAndCannotForgeAMint`
(internal/app): a server boots with a `ListenAddr` + one `ProducerGrant`. An
EXTERNAL `nats.Connect(liveFabric.Addr(), UserInfo("prodA","pwA"))` client opens a
JetStream context and:

- **Allowed path:** publishes a real `ledger.ClaimRecord` to its own claim subtree
  (`packets.session.default.events.ledger.claim.work`). The host's spawned consumer
  drains it through a confirming verifier and MINTS — `log.Balance()` reaches 1.
  This exercises authenticate → grant-confined publish → consume → verify → mint
  over the actual TCP socket, not an in-process shortcut.
- **Denied path:** the same client's publish to the MINTED subtree
  (`…minted.catch`) is refused with a permissions violation (returned as a publish
  error) — proving a producer can never forge a catch, end to end through the live
  server's wiring, not just at the fabric unit level.

The verifier is the confirming stub (the real CageVerifier is locked by the
equivalence lock elsewhere) because this test's subject is the AUTHENTICATED
TRANSPORT + the mint wiring, not cage materialization.

## Verdict

Full repo green with `-race`. The producer-auth boundary is now proven end to end:
an authenticated external producer's claim mints, and a forged mint is denied,
over a real socket through the live server. The "tested by contract" caveat from
R81/R82 is discharged.

## New clashes opened / resolved

Resolved: the producer-auth boundary holds end-to-end against a real external NATS
client (allowed claim mints; forged mint denied), not only by composition of
unit-tested parts. No new clash.
</content>
