package fabric_test

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
)

func startListening(t *testing.T, grants ...fabric.ProducerGrant) *fabric.Fabric {
	t.Helper()
	f, err := fabric.StartListening(context.Background(), t.TempDir(), "127.0.0.1:0", grants...)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// producerPublish connects as a credentialed producer over the listen socket and
// JetStream-publishes to subject, returning the publish error (a denied subject
// gets no ack → error). A JS publish needs to receive its ack on the client's
// _INBOX, so an ALLOWED publish only succeeds if the producer is also granted
// subscribe on _INBOX.> — that narrow grant (and nothing broader) is what makes
// the allowed-publish assertion pass while the subscribe-confinement test holds.
func producerPublish(t *testing.T, f *fabric.Fabric, user, pass, subject string) error {
	t.Helper()
	pc, err := nats.Connect(f.Addr(), nats.UserInfo(user, pass))
	require.NoError(t, err)
	defer pc.Close()
	pjs, err := pc.JetStream()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = pjs.Publish(subject, []byte("x"), nats.Context(ctx))
	return err
}

func TestStartListening_confinesAProducerToItsOwnClaimSubtree(t *testing.T) {
	t.Parallel()
	f := startListening(t, fabric.ProducerGrant{User: "prodA", Pass: "pwA", Session: "A", Instance: "i"})

	// Allowed: its own claim subtree — claims are what a producer may publish.
	assert.NoError(t, producerPublish(t, f, "prodA", "pwA",
		fabric.EventSubject("A", "i", fabric.StatusClaim, "diff")))

	// Denied: minting is reserved to the host (the verifier mints, not the producer).
	assert.Error(t, producerPublish(t, f, "prodA", "pwA",
		fabric.EventSubject("A", "i", fabric.StatusMinted, "catch")))

	// Denied: another session's subtree (no cross-session forgery).
	assert.Error(t, producerPublish(t, f, "prodA", "pwA",
		fabric.EventSubject("B", "i", fabric.StatusClaim, "diff")))
}

func TestStartListening_deniesAProducerSubscribingBeyondItsInbox(t *testing.T) {
	t.Parallel()
	f := startListening(t, fabric.ProducerGrant{User: "prodA", Pass: "pwA", Session: "A", Instance: "i"})

	// A producer that could subscribe to the whole fabric could exfiltrate every
	// session's economy. Its subscribe grant must be confined to its own reply
	// inbox; attempting to read packets.> must raise a permissions violation.
	violations := make(chan error, 4)
	pc, err := nats.Connect(f.Addr(), nats.UserInfo("prodA", "pwA"),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, e error) { violations <- e }))
	require.NoError(t, err)
	defer pc.Close()

	_, err = pc.SubscribeSync("packets.>")
	require.NoError(t, err) // the subscribe call returns nil; the violation is async
	require.NoError(t, pc.Flush())

	select {
	case e := <-violations:
		assert.Contains(t, e.Error(), "Permissions Violation")
	case <-time.After(2 * time.Second):
		t.Fatal("expected a permissions violation for the cross-fabric subscribe")
	}
}

func TestStartListening_rejectsAWrongCredentialConnection(t *testing.T) {
	t.Parallel()
	f := startListening(t, fabric.ProducerGrant{User: "prodA", Pass: "pwA", Session: "A", Instance: "i"})

	_, err := nats.Connect(f.Addr(), nats.UserInfo("prodA", "wrong"))
	assert.Error(t, err, "a wrong credential must be rejected at connect")
}

func TestStartListening_rejectsAMalformedListenAddr(t *testing.T) {
	t.Parallel()
	for _, addr := range []string{"no-port", "127.0.0.1:notaport"} {
		_, err := fabric.StartListening(context.Background(), t.TempDir(), addr)
		assert.Error(t, err, "addr %q must be rejected", addr)
	}
}

func TestStartListening_deniesAnAnonymousExternalClientHostPrivileges(t *testing.T) {
	t.Parallel()
	f := startListening(t, fabric.ProducerGrant{User: "prodA", Pass: "pwA", Session: "A", Instance: "i"})

	// NoAuthUser=hostUser maps the credential-less in-process connection to the
	// full-perms host. An EXTERNAL socket client presenting no credentials must
	// NOT inherit that mapping: if it did, an unauthenticated client could mint.
	// The host identity is confined to in-process connections, so an anonymous
	// external connect either fails outright or, if it connects, cannot publish a
	// minted subject.
	nc, err := nats.Connect(f.Addr())
	if err == nil {
		defer nc.Close()
		js, jerr := nc.JetStream()
		require.NoError(t, jerr)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, perr := js.Publish(fabric.EventSubject("A", "i", fabric.StatusMinted, "catch"),
			[]byte("m"), nats.Context(ctx))
		assert.Error(t, perr, "an anonymous external client must not be able to mint")
		return
	}
	assert.Error(t, err, "an anonymous external client must not connect as the host")
}

func TestStartListening_keepsMintingAvailableToTheInProcessHost(t *testing.T) {
	t.Parallel()
	f := startListening(t, fabric.ProducerGrant{User: "prodA", Pass: "pwA", Session: "A", Instance: "i"})

	_, err := f.Publish(context.Background(),
		fabric.EventSubject("A", "i", fabric.StatusMinted, "catch"), []byte("m"))
	assert.NoError(t, err, "the in-process host retains full minting publish")
}
