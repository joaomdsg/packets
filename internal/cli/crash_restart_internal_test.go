package cli

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/socket"
)

// A parked session's identity must survive a SUDDEN crash, not just a graceful
// shutdown: the KV Put is durable on write (JetStream FileStorage), so a
// SIGKILL cannot lose it. Build the real server, let it park the default
// session, SIGKILL it (no graceful teardown), then reopen the store dir cold
// and assert the parked record is still there.
func TestServer_parkedIdentitySurvivesASIGKILL(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the server binary; skipped under -short")
	}
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "packets")
	build := exec.Command("go", "build", "-o", bin, "github.com/joaomdsg/packets/cmd/packets")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "build server: %s", out)

	wd, err := os.Getwd()
	require.NoError(t, err)
	repoRoot := filepath.Join(wd, "..", "..") // internal/cli -> repo root, a real git repo
	ledger := filepath.Join(tmp, "catches")
	logPath := filepath.Join(tmp, "server.log")
	logFile, err := os.Create(logPath)
	require.NoError(t, err)
	defer logFile.Close()

	srv := exec.Command(bin,
		"-repo", repoRoot, // registers the default (prompt-authoring) session
		"-addr", freePort(t),
		"-ledger", ledger,
		"-park-idle", "300ms", // park the idle default quickly
		"-park-interval", "100ms",
	)
	srv.Stdout, srv.Stderr = logFile, logFile
	require.NoError(t, srv.Start())
	killed := false
	defer func() {
		if !killed {
			_ = srv.Process.Kill()
		}
	}()

	// Wait for boot, then long enough that the idle sweep has parked default.
	require.Eventually(t, func() bool {
		b, _ := os.ReadFile(logPath)
		return len(b) > 0 && containsStr(string(b), "serving")
	}, 20*time.Second, 100*time.Millisecond, "server did not boot")
	time.Sleep(1500 * time.Millisecond) // park-idle 300ms + a few 100ms sweeps

	// Sudden crash: SIGKILL, no graceful shutdown/flush.
	require.NoError(t, srv.Process.Signal(syscall.SIGKILL))
	_, _ = srv.Process.Wait()
	killed = true

	// Reopen the store dir cold — as a restart would — and assert the record survived.
	f, err := fabric.Start(context.Background(), ledger+"-fabric")
	require.NoError(t, err)
	defer f.Close()
	r, err := socket.OpenParkedRegistry(f)
	require.NoError(t, err)
	entries, err := r.List()
	require.NoError(t, err)

	addrs := make([]string, 0, len(entries))
	for _, e := range entries {
		addrs = append(addrs, e.Addr)
	}
	require.Contains(t, addrs, "default", "the parked identity must survive a SIGKILL on disk (durable KV Put)")
}

// freePort reserves an ephemeral port and releases it, returning the address
// for the subprocess to bind — avoids a hardcoded port that could conflict.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return l.Addr().String()
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
