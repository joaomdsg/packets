// Package settle turns a harness turn's work into a git revision. Its first
// responsibility is the change guard: a revision is minted only
// when the working tree actually changed, so a no-edit turn (a question:
// answered with no code, or an edit reverted) does not error or create an
// empty revision.
package settle

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// errNoStagedChange signals that, after staging, the index matches HEAD: a
// net-revert turn with nothing to commit (not a failure).
var errNoStagedChange = errors.New("settle: no staged change")

// SecretHit reports a likely secret introduced by the staged change, anchored
// to the file and 1-based line the added content appears on, with the name of
// the rule that matched.
type SecretHit struct {
	File string
	Line int
	Rule string
}

// Result reports what Settle did with a turn's work.
type Result struct {
	Committed bool        // whether a new revision was minted
	SHA       string      // the new commit's full SHA (empty when nothing was committed)
	Secrets   []SecretHit // non-empty when the commit was BLOCKED by a detected secret
	// Artifacts lists staged BINARY files in a minted revision — surfaced, never
	// dropped: they pollute the review diff and can't be line-reviewed or
	// secret-scanned, so the reviewer is told about them. Empty unless Committed.
	Artifacts []string
}

// Settle commits the working tree of the git repo at repoDir as a new revision
// — but ONLY when the tree actually changed. A turn that changed nothing mints
// no revision and returns no error, rather than letting an unconditional
// `git add -A && git commit` fail with "nothing to commit".
func Settle(ctx context.Context, repoDir, message string) (Result, error) {
	status, err := git(ctx, repoDir, "status", "--porcelain")
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(status) == "" {
		return Result{}, nil // no-edit turn: nothing to revise
	}

	// Staging keeps `git add -A` (honoring .gitignore). We do NOT silently
	// exclude artifacts — that risks dropping a file the agent meant to commit.
	// Instead, binary files are SURFACED below (Result.Artifacts) so the reviewer
	// sees them. Further artifact heuristics (large text files, size thresholds,
	// extension globs) are a deferred policy decision.
	if _, err := git(ctx, repoDir, "add", "-A"); err != nil {
		return Result{}, err
	}

	// The porcelain pre-check is necessary but not sufficient: a turn that
	// staged a change then reverted the worktree to HEAD's content shows a
	// non-empty porcelain ("MM file") yet, once `add -A` restages the worktree,
	// the index matches HEAD again — nothing to commit. Committing anyway fails
	// with "nothing to commit" and surfaces as an error, the exact desync this
	// guard exists to absorb. So gate the commit on the index actually differing
	// from HEAD. `git diff --cached --quiet` exits 0 when they match (no
	// revision), 1 when they differ (commit), and >1 only on real error.
	if err := indexDiffersFromHEAD(ctx, repoDir); err != nil {
		if errors.Is(err, errNoStagedChange) {
			return Result{}, nil // net-revert turn: nothing to revise
		}
		return Result{}, err
	}

	// Scan the lines this revision ADDS for secrets BEFORE committing, so a
	// secret never enters git history. A hit is a surfaced block for the
	// reviewer, not an infra error: no commit, no error.
	// Force canonical diff output regardless of the repo/user git config: a
	// hostile or merely customized config (color.diff=always injects ANSI
	// escapes so an added line no longer starts with "+"; diff.noprefix /
	// diff.mnemonicPrefix change the "+++ b/<path>" header; diff.external swaps
	// in a foreign formatter) would otherwise let a secret slip past the parser
	// unscanned and into history. --no-color/--no-ext-diff and explicit
	// src/dst prefixes pin the format scanStagedDiff parses.
	diff, err := git(ctx, repoDir, "diff", "--cached", "--no-color", "--no-ext-diff", "--src-prefix=a/", "--dst-prefix=b/")
	if err != nil {
		return Result{}, err
	}
	if hits := scanStagedDiff(diff, secretRules); len(hits) > 0 {
		return Result{Committed: false, Secrets: hits}, nil
	}

	// Surface (don't drop) staged binary files so the reviewer is aware of
	// unreviewable artifacts in this revision.
	artifacts, err := stagedBinaryFiles(ctx, repoDir)
	if err != nil {
		return Result{}, err
	}

	if _, err := git(ctx, repoDir, "commit", "-m", message); err != nil {
		return Result{}, err
	}
	sha, err := git(ctx, repoDir, "rev-parse", "HEAD")
	if err != nil {
		return Result{}, err
	}
	return Result{Committed: true, SHA: strings.TrimSpace(sha), Artifacts: artifacts}, nil
}

// stagedBinaryFiles returns the paths of staged files git treats as binary and
// that are PRESENT in the revision (added or modified) — a deleted binary is
// not an artifact polluting the diff, so it is excluded (--diff-filter=d,
// lowercase = exclude deletions). In `git diff --cached --numstat` a binary
// file's added/deleted columns are a literal "-" (rather than line counts);
// --no-renames keeps paths un-munged (a rename shows as delete+add, consistent
// with the diff package). -z makes the output NUL-terminated with RAW,
// unquoted paths (otherwise core.quotePath wraps non-ASCII/control-char paths
// in "..." with C-escapes, surfacing an unusable mangled path); each record is
// `add<TAB>del<TAB>path`.
func stagedBinaryFiles(ctx context.Context, repoDir string) ([]string, error) {
	out, err := git(ctx, repoDir, "diff", "--cached", "--numstat", "--no-renames", "--diff-filter=d", "-z")
	if err != nil {
		return nil, err
	}
	var bin []string
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		fields := strings.SplitN(rec, "\t", 3)
		if len(fields) == 3 && fields[0] == "-" && fields[1] == "-" {
			bin = append(bin, fields[2])
		}
	}
	return bin, nil
}

// indexDiffersFromHEAD reports whether the staged index differs from HEAD.
// It returns nil when there is a staged change to commit, errNoStagedChange
// when the index matches HEAD (exit 0), and a wrapped error on any real
// failure (exit >1, e.g. not a repo). When there is no HEAD yet (an unborn
// branch with staged content), `diff --cached --quiet` exits 1 — a diff vs the
// empty tree — so a first commit proceeds, which is correct.
func indexDiffersFromHEAD(ctx context.Context, repoDir string) error {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--quiet")
	cmd.Dir = repoDir
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err == nil {
		return errNoStagedChange // exit 0: index == HEAD, nothing to commit
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit.ExitCode() == 1 {
		return nil // exit 1: index differs from HEAD, commit it
	}
	return fmt.Errorf("git diff --cached --quiet: %w: %s", err, strings.TrimSpace(errBuf.String()))
}

// git runs a git subcommand in repoDir and returns its stdout, wrapping any
// failure with the command and stderr for diagnosis.
func git(ctx context.Context, repoDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = repoDir
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String(), nil
}

type secretRule struct {
	name string
	re   *regexp.Regexp
}

// secretRules is a finite, HIGH-CONFIDENCE default set chosen for distinctive
// structure (low false-positive). It covers a generic secret-named assignment
// plus a handful of well-known provider token formats. Further expansion —
// entropy/randomness detection, an allowlist for test fixtures, more providers,
// false-positive tuning — remains a deferred policy decision (see RISKS.md
// "settle-git-add-all-pollution"). It is a package-level var so it can be
// extended without touching the scan logic.
var secretRules = []secretRule{
	{"private-key", regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)},
	{"aws-access-key-id", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"secret-assignment", regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|access[_-]?key|secret[_-]?key)\s*[:=]\s*["']?[A-Za-z0-9/+_-]{16,}`)},
	{"github-token", regexp.MustCompile(`(gh[pousr]_[0-9A-Za-z]{36}|github_pat_[0-9A-Za-z_]{22,})`)},
	{"google-api-key", regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`)},
	{"slack-token", regexp.MustCompile(`xox[baprs]-[0-9A-Za-z-]{10,}`)},
	{"stripe-secret-key", regexp.MustCompile(`(sk|rk)_live_[0-9A-Za-z]{24,}`)},
}

// scanStagedDiff parses a unified `git diff --cached` and returns a SecretHit
// for every rule that matches a line the diff ADDS (never pre-existing or
// removed content). It is a pure function over the diff text — the I/O (running
// git) lives in Settle — so the parsing/matching is exercised through Settle's
// real-git tests. File comes from the `+++ b/<path>` header; the 1-based new
// line number is tracked through each hunk's `@@ ... +start,len @@`.
func scanStagedDiff(diff string, rules []secretRule) []SecretHit {
	var hits []SecretHit
	var file string
	var newLine int
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			file = strings.TrimPrefix(strings.TrimPrefix(line, "+++ "), "b/")
		case strings.HasPrefix(line, "@@"):
			newLine = hunkNewStart(line)
		case strings.HasPrefix(line, "+"): // an added line (not the "+++" header, handled above)
			text := line[1:]
			for _, r := range rules {
				if r.re.MatchString(text) {
					hits = append(hits, SecretHit{File: file, Line: newLine, Rule: r.name})
				}
			}
			newLine++
		case strings.HasPrefix(line, "-"), strings.HasPrefix(line, "\\"):
			// removed line / "\ No newline at end of file": no new-line advance.
		default:
			// context line (leading space) or diff/index headers: advance only
			// for actual context within a hunk.
			if strings.HasPrefix(line, " ") {
				newLine++
			}
		}
	}
	return hits
}

// hunkNewStart extracts the new-file start line from a hunk header like
// "@@ -a,b +c,d @@" (returns c), or 0 if it can't be parsed.
func hunkNewStart(hunk string) int {
	plus := strings.IndexByte(hunk, '+')
	if plus < 0 {
		return 0
	}
	n := 0
	for i := plus + 1; i < len(hunk); i++ {
		c := hunk[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
