// Package fabric is the Packets orchestration spine (Phase 0): an embedded
// NATS/JetStream server wrapping the authoritative, append-only event log.
// Every orchestration event is published here first; projections (the board,
// the ledger) are rebuilt from this log, never written ahead of it.
package fabric

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

// streamName is the single authoritative stream; subject "packets.>" captures
// the whole subject taxonomy (packets.session.<sid>.events.<inst>.<status>.<kind>).
const streamName = "PACKETS"

// Event is one stored entry in the log, with the JetStream sequence that fixes
// its global order.
type Event struct {
	Subject string
	Seq     uint64
	Data    []byte
}

// Fabric is a running embedded JetStream log rooted at a storage dir.
type Fabric struct {
	ns *server.Server
	nc *nats.Conn
	js nats.JetStreamContext
}

// Start boots an in-process JetStream server storing the log under dir and
// ensures the authoritative stream exists. The server listens on no TCP port;
// connections are in-process only.
func Start(ctx context.Context, dir string) (*Fabric, error) {
	ns, err := server.NewServer(&server.Options{
		JetStream:  true,
		StoreDir:   dir,
		DontListen: true,
	})
	if err != nil {
		return nil, fmt.Errorf("fabric: new server: %v", err)
	}
	ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		return nil, fmt.Errorf("fabric: server not ready")
	}

	nc, err := nats.Connect("", nats.InProcessServer(ns))
	if err != nil {
		ns.Shutdown()
		return nil, fmt.Errorf("fabric: connect: %v", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		ns.Shutdown()
		return nil, fmt.Errorf("fabric: jetstream: %v", err)
	}

	if _, err := js.StreamInfo(streamName); err != nil {
		if _, err := js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"packets.>"},
			Storage:  nats.FileStorage,
		}); err != nil {
			nc.Close()
			ns.Shutdown()
			return nil, fmt.Errorf("fabric: add stream: %v", err)
		}
	}

	return &Fabric{ns: ns, nc: nc, js: js}, nil
}

// Close drains the connection and shuts the embedded server down.
func (f *Fabric) Close() error {
	f.nc.Close()
	f.ns.Shutdown()
	f.ns.WaitForShutdown()
	return nil
}

// Publish appends data on subject to the log and returns its assigned sequence.
func (f *Fabric) Publish(ctx context.Context, subject string, data []byte) (uint64, error) {
	ack, err := f.js.Publish(subject, data, nats.Context(ctx))
	if err != nil {
		return 0, fmt.Errorf("fabric: publish %s: %v", subject, err)
	}
	return ack.Sequence, nil
}

// Replay returns every stored event from the first sequence onward, in order.
func (f *Fabric) Replay(ctx context.Context) ([]Event, error) {
	info, err := f.js.StreamInfo(streamName, nats.Context(ctx))
	if err != nil {
		return nil, fmt.Errorf("fabric: stream info: %v", err)
	}
	var events []Event
	// Empty stream reports FirstSeq 0; valid sequences start at 1, so an
	// unguarded loop would GetMsg(0) and fail with "bad request".
	for seq := info.State.FirstSeq; seq >= 1 && seq <= info.State.LastSeq; seq++ {
		msg, err := f.js.GetMsg(streamName, seq, nats.Context(ctx))
		if err != nil {
			return nil, fmt.Errorf("fabric: get msg %d: %v", seq, err)
		}
		events = append(events, Event{Subject: msg.Subject, Seq: msg.Sequence, Data: msg.Data})
	}
	return events, nil
}
