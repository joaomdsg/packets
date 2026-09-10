package socket_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/socket"
)

func TestParkedRegistry_putThenListRoundTripsEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	e1 := socket.ParkedEntry{Addr: "127.0.0.1:6222", Dir: "/d1", Session: "s1", Instance: "i1"}
	e2 := socket.ParkedEntry{Addr: "127.0.0.1:6223", Dir: "/d2", Session: "s2", Instance: "i2"}
	require.NoError(t, r.Put(e1))
	require.NoError(t, r.Put(e2))

	got, err := r.List()
	require.NoError(t, err)
	assert.ElementsMatch(t, []socket.ParkedEntry{e1, e2}, got)
}

func TestParkedRegistry_deleteRemovesOnlyThatEntry(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	e1 := socket.ParkedEntry{Addr: "127.0.0.1:6222", Dir: "/d1", Session: "s1", Instance: "i1"}
	e2 := socket.ParkedEntry{Addr: "127.0.0.1:6223", Dir: "/d2", Session: "s2", Instance: "i2"}
	require.NoError(t, r.Put(e1))
	require.NoError(t, r.Put(e2))

	require.NoError(t, r.Delete(e1.Addr))

	got, err := r.List()
	require.NoError(t, err)
	assert.Equal(t, []socket.ParkedEntry{e2}, got)
}

func TestParkedRegistry_deleteOfAbsentAddrIsNotAnError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	assert.NoError(t, r.Delete("127.0.0.1:9999"))
}

func TestParkedRegistry_listOnEmptyBucketIsEmptyNotError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	got, err := r.List()
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestParkedRegistry_entriesSurviveFabricRestartOnSameDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	e := socket.ParkedEntry{Addr: "127.0.0.1:6222", Dir: "/d1", Session: "s1", Instance: "i1"}
	require.NoError(t, r.Put(e))
	require.NoError(t, f.Close())

	f2, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f2.Close() })

	r2, err := socket.OpenParkedRegistry(f2)
	require.NoError(t, err)

	got, err := r2.List()
	require.NoError(t, err)
	assert.Equal(t, []socket.ParkedEntry{e}, got)
}

func TestParkedRegistry_persistedValueHasNoGrantOrCredentialFields(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)

	e := socket.ParkedEntry{Addr: "127.0.0.1:6222", Dir: "/d1", Session: "s1", Instance: "i1"}
	require.NoError(t, r.Put(e))

	kv, err := f.OpenKV("parked_sockets")
	require.NoError(t, err)
	keys, err := kv.Keys()
	require.NoError(t, err)
	require.Len(t, keys, 1)
	entry, err := kv.Get(keys[0])
	require.NoError(t, err)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(entry.Value(), &raw))
	assert.ElementsMatch(t, []string{"Addr", "Dir", "Session", "Instance"}, keysOf(raw))
}

func keysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
