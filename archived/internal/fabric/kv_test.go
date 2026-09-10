package fabric_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
)

func TestFabric_openKVRoundTripsPutGetDelete(t *testing.T) {
	t.Parallel()

	f, err := fabric.Start(context.Background(), t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })

	kv, err := f.OpenKV("prefs")
	require.NoError(t, err)

	_, err = kv.Put("theme", []byte("dark"))
	require.NoError(t, err)

	entry, err := kv.Get("theme")
	require.NoError(t, err)
	require.Equal(t, []byte("dark"), entry.Value())

	require.NoError(t, kv.Delete("theme"))
	_, err = kv.Get("theme")
	require.Error(t, err)
}

func TestFabric_openKVSurvivesRestartOnSameDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	f, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)

	kv, err := f.OpenKV("prefs")
	require.NoError(t, err)
	_, err = kv.Put("theme", []byte("dark"))
	require.NoError(t, err)

	require.NoError(t, f.Close())

	f2, err := fabric.Start(context.Background(), dir)
	require.NoError(t, err)
	t.Cleanup(func() { f2.Close() })

	kv2, err := f2.OpenKV("prefs")
	require.NoError(t, err)

	entry, err := kv2.Get("theme")
	require.NoError(t, err)
	require.Equal(t, []byte("dark"), entry.Value())
}
