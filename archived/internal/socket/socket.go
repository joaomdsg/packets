// Package socket manages an addr's live attachment to the host-owned fabric
// for the addr's whole lifetime: warm while attached, parked to a resumable
// ticket when idle. The addr is the durable identity; the socket is not. The
// fabric is the network, stood up and owned by the host; a socket only rides
// its subtree — it owns WHEN its attachment runs (Open/Resume start it,
// Park/Close stop it), never the fabric's lifecycle.
package socket

import (
	"context"
	"fmt"
	"sync"

	"github.com/joaomdsg/packets/internal/fabric"
)

// Attach is an addr's long-running work on the fabric — typically the durable
// consumer that accepts the addr's sends. It runs until its context is
// cancelled and returns when it is. The socket owns when it runs; the caller
// supplies what it does, so the low-level socket never needs the app's logic.
type Attach func(ctx context.Context) error

// Socket is one addr's attachment onto the shared, host-owned fabric: it holds
// the addr's Attach and runs it under a socket-owned, cancelable context.
type Socket struct {
	mu     sync.Mutex
	fab    *fabric.Fabric
	addr   string
	attach Attach
	cancel context.CancelFunc // cancels the running attach; nil when not running
	done   chan struct{}      // closed when the running attach goroutine exits
	ticket *Ticket            // non-nil while parked
	closed bool
}

// Ticket is a single-use, in-memory handle on a parked addr, carrying the
// shared fabric, the addr, and its Attach so Resume can re-attach without
// rebuilding the network. It is redeemable exactly once.
type Ticket struct {
	mu     sync.Mutex
	fab    *fabric.Fabric
	addr   string
	attach Attach
	used   bool
}

// Open attaches addr's endpoint onto the host-owned fabric fab and starts its
// attach running. addr is the durable owner/repo subject token (never a
// host:port); fab is the shared network the host already stood up. The attach
// runs under a cancelable child of ctx, so the caller's ctx bounds its life and
// Park/Close can stop it independently.
func Open(ctx context.Context, fab *fabric.Fabric, addr string, attach Attach) (*Socket, error) {
	if fab == nil {
		return nil, fmt.Errorf("socket: open %q: nil fabric", addr)
	}
	if addr == "" {
		return nil, fmt.Errorf("socket: open: empty addr")
	}
	if attach == nil {
		return nil, fmt.Errorf("socket: open %q: nil attach", addr)
	}
	s := &Socket{fab: fab, addr: addr, attach: attach}
	s.start(ctx)
	return s, nil
}

// start launches the attach under a fresh cancelable child of ctx, recording
// the cancel func and a done channel closed when the goroutine exits. It needs
// no lock: it runs only from Open and Resume, before the *Socket is handed
// back, so the socket is not yet shared with any other goroutine.
func (s *Socket) start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	attach := s.attach
	go func() {
		defer close(done)
		_ = attach(runCtx)
	}()
}

// stop cancels the running attach and waits for its goroutine to exit, so no
// two attaches ever run the same durable at once. s.mu held. Safe if already
// stopped (cancel/done nil).
func (s *Socket) stop() {
	if s.cancel == nil {
		return
	}
	s.cancel()
	<-s.done
	s.cancel = nil
	s.done = nil
}

// Addr is the addr claimed by Open, stable across park/resume.
func (s *Socket) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

// Fabric is the shared, host-owned fabric the addr rides. It stays valid
// across Park and after Close: the socket never owns the fabric's lifecycle.
func (s *Socket) Fabric() *fabric.Fabric {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.fab
}

// Park stops the addr's attach (cancelling it and waiting for it to exit) and
// returns a single-use Ticket that Resume redeems to re-attach. The shared
// fabric is left untouched — park releases the addr's work, not the network.
func (s *Socket) Park() (*Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, fmt.Errorf("socket: park %q: already closed", s.addr)
	}
	if s.ticket != nil {
		return nil, fmt.Errorf("socket: park %q: already parked", s.addr)
	}
	s.stop()
	t := &Ticket{fab: s.fab, addr: s.addr, attach: s.attach}
	s.ticket = t
	return t, nil
}

// Resume re-attaches a parked ticket's addr as a warm Socket on the same shared
// fabric, restarting its attach under a fresh cancelable child of ctx. It never
// creates a fabric. The ticket is consumed; redeeming it a second time, or
// after the parking socket was closed, errors.
func Resume(ctx context.Context, t *Ticket) (*Socket, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.used {
		return nil, fmt.Errorf("socket: resume: ticket already used")
	}
	t.used = true
	s := &Socket{fab: t.fab, addr: t.addr, attach: t.attach}
	s.start(ctx)
	return s, nil
}

// Close releases the socket: it stops the running attach (cancelling and
// waiting for it to exit) and, while parked, invalidates the outstanding ticket
// so a stale ticket can never resurrect an endpoint the owner already tore
// down. The host-owned fabric is never closed. Close is idempotent.
func (s *Socket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	s.stop()
	if s.ticket != nil {
		t := s.ticket
		s.ticket = nil
		t.mu.Lock()
		t.used = true
		t.mu.Unlock()
	}
	return nil
}
