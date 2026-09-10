package socket

import (
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/nats-io/nats.go"

	"github.com/joaomdsg/packets/internal/fabric"
)

// ParkedEntry is the non-secret routing identity of a parked socket.
// Grants and credentials are re-derived from boot config at resume and
// must never be added here.
type ParkedEntry struct {
	Addr     string
	Dir      string
	Session  string
	Instance string
}

// ParkedRegistry is a KV-backed durable record of parked sockets, keyed by
// addr, surviving a restart of the fabric on the same store dir.
type ParkedRegistry struct {
	kv nats.KeyValue
}

// OpenParkedRegistry opens (creating if absent) the "parked_sockets" KV
// bucket on f.
func OpenParkedRegistry(f *fabric.Fabric) (*ParkedRegistry, error) {
	kv, err := f.OpenKV("parked_sockets")
	if err != nil {
		return nil, err
	}
	return &ParkedRegistry{kv: kv}, nil
}

func parkedKey(addr string) string {
	return hex.EncodeToString([]byte(addr))
}

// Put upserts e, keyed by e.Addr.
func (r *ParkedRegistry) Put(e ParkedEntry) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = r.kv.Put(parkedKey(e.Addr), b)
	return err
}

// List returns all parked entries, or an empty slice with a nil error when
// none are parked.
func (r *ParkedRegistry) List() ([]ParkedEntry, error) {
	keys, err := r.kv.Keys()
	if errors.Is(err, nats.ErrNoKeysFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries := make([]ParkedEntry, 0, len(keys))
	for _, k := range keys {
		v, err := r.kv.Get(k)
		if err != nil {
			return nil, err
		}
		var e ParkedEntry
		if err := json.Unmarshal(v.Value(), &e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Delete removes the parked entry for addr. Deleting an absent addr is not
// an error.
func (r *ParkedRegistry) Delete(addr string) error {
	err := r.kv.Delete(parkedKey(addr))
	if errors.Is(err, nats.ErrKeyNotFound) {
		return nil
	}
	return err
}
