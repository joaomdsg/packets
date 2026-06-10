// Package ingest is the trusted host-side intake of an untrusted producer's git
// objects: it validates a producer-supplied git bundle and unbundles it into a
// per-producer ref namespace in the host object store, so a later cage
// verification can resolve the claim's revisions WITHOUT the host ever fetching
// from a producer-controlled URL (no egress, no SSRF — council round 38).
//
// It is fail-closed and namespace-confined: every ingested ref is forced under
// refs/producers/<producerID>/*, so no producer-chosen ref name can touch the
// host's own refs or another producer's. A malformed, oversized, or
// wrongly-named submission is rejected and writes nothing.
package ingest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// ErrBundleInvalid marks a producer payload that is not a valid, unbundleable git
// bundle (garbage bytes, a truncated/corrupt pack, or missing prerequisites).
var ErrBundleInvalid = errors.New("ingest: producer bundle is invalid")

// ErrCapExceeded marks a producer bundle past the byte cap — rejected before any
// git work, so a pack/bundle bomb costs the host nothing.
var ErrCapExceeded = errors.New("ingest: producer bundle exceeds the size cap")

// ErrBadProducerID marks a producer id that is not a safe single ref segment —
// it could otherwise escape its namespace via the refspec (path traversal or
// extra ref segments) and write the host's or another producer's refs.
var ErrBadProducerID = errors.New("ingest: producer id is not a safe ref segment")

// safeProducerID is the trusted ref-segment alphabet: it cannot contain '/', '.'
// runs that traverse, whitespace, or git refspec metacharacters, so it can only
// ever name one segment under refs/producers/.
var safeProducerID = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// IngestProducerObjects validates a producer's git bundle and unbundles it into
// refs/producers/<producerID>/* in the host store (an existing git repo). The
// order is cheapest-guard-first: id validation, then the byte cap (before any
// git work), then bundle verification, then a forced-refspec fetch that confines
// every ref to the producer namespace. Any failure writes nothing to the store.
func IngestProducerObjects(ctx context.Context, store, producerID string, bundle []byte, maxBytes int64) error {
	// A ".." anywhere is a refspec range/traversal metacharacter (even when the
	// alphabet check passes, e.g. "a..b"); a lone "." is the current-dir segment.
	// Reject both up front so the failure is an honest bad-id, not a downstream
	// "invalid bundle" once git refuses the refspec.
	if !safeProducerID.MatchString(producerID) || producerID == "." || strings.Contains(producerID, "..") {
		return fmt.Errorf("ingest: producer id %q: %w", producerID, ErrBadProducerID)
	}
	if maxBytes > 0 && int64(len(bundle)) > maxBytes {
		return fmt.Errorf("ingest: bundle is %d bytes (cap %d): %w", len(bundle), maxBytes, ErrCapExceeded)
	}

	tmp, err := os.CreateTemp("", "packets-ingest-*.bundle")
	if err != nil {
		return fmt.Errorf("ingest: temp bundle: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(bundle); err != nil {
		tmp.Close()
		return fmt.Errorf("ingest: write bundle: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("ingest: close bundle: %w", err)
	}

	// Verify the bundle is well-formed before letting it touch the store; fold the
	// git stderr into the message text (one %w, on the sentinel).
	if out, err := git(ctx, "", "bundle", "verify", tmp.Name()); err != nil {
		return fmt.Errorf("ingest: bundle verify (%s): %w", strings.TrimSpace(out), ErrBundleInvalid)
	}

	// Force every bundle ref refs/X to refs/producers/<id>/X: a bundle naming
	// refs/heads/main lands at refs/producers/<id>/heads/main, never the host's
	// own refs/heads/main. --end-of-options guards the temp path; the refspec is
	// the namespacing.
	refspec := "refs/*:refs/producers/" + producerID + "/*"
	if out, err := git(ctx, store, "fetch", "--no-tags", "--no-write-fetch-head",
		"--end-of-options", tmp.Name(), refspec); err != nil {
		return fmt.Errorf("ingest: unbundle into namespace (%s): %w", strings.TrimSpace(out), ErrBundleInvalid)
	}
	return nil
}

// git runs a git command (in dir, or the process cwd when dir is empty) and
// returns its combined output plus an error carrying that output on failure.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("git %s: %v", strings.Join(args, " "), err)
	}
	return out.String(), nil
}
