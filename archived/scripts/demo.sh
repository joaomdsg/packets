#!/usr/bin/env bash
# demo.sh — an honest, end-to-end walkthrough of the packets MVP on a REAL
# local repo: compose -> forward -> hold -> inspect -> deliver.
#
# It builds the packets binary, constructs a throwaway git repo with three
# real revisions (a base with an under-tested boundary, a fix that catches
# it, and a fix with a broken build), then runs every piece of the pipeline
# that's fully scriptable (the gate/catch mechanics via `verify-catch`, the
# adversarial mode via `probe`) before booting the real server and printing
# the browser steps for the interactive part (composing, watching a hold,
# inspecting, delivering) — those are genuinely UI actions, not curl calls.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

echo "== building packets =="
go build -o "$WORK/packets" "$ROOT/cmd/packets"

echo
echo "== constructing a real scratch repo =="
DEMO="$WORK/demo-repo"
mkdir -p "$DEMO"
cd "$DEMO"
git init -q
git config user.email demo@packets.local
git config user.name "packets demo"

cat > go.mod <<'EOF'
module demo

go 1.21
EOF
cat > adult.go <<'EOF'
package demo

func IsAdult(age int) bool {
	return age >= 18
}
EOF
cat > adult_test.go <<'EOF'
package demo

import "testing"

// A weak test: it only ever asks about an age well past the boundary, so a
// mutated >= (e.g. >) still returns the same answer and survives undetected.
func TestIsAdult(t *testing.T) {
	if !IsAdult(20) {
		t.Fatal("20 should be an adult")
	}
}
EOF
git add -A && git commit -qm "base: weak boundary test"
BASE=$(git rev-parse HEAD)

cat > adult_test.go <<'EOF'
package demo

import "testing"

// The strengthened test pins BOTH sides of the age-18 boundary — a mutated
// >= no longer produces the same answer at 18 or 17, so it can't survive.
func TestIsAdult(t *testing.T) {
	if !IsAdult(20) {
		t.Fatal("20 should be an adult")
	}
	if !IsAdult(18) {
		t.Fatal("18 should be an adult")
	}
	if IsAdult(17) {
		t.Fatal("17 should not be an adult")
	}
}
EOF
git add -A && git commit -qm "fix: pin both sides of the age-18 boundary"
GOODFIX=$(git rev-parse HEAD)

git checkout -q "$BASE"
cat > adult.go <<'EOF'
package demo

func IsAdult(age int) bool {
	return age >= 18 this is not valid go
}
EOF
git add -A && git commit -qm "fix: broken build"
BADFIX=$(git rev-parse HEAD)
git checkout -q "$GOODFIX"

echo "  base:     $BASE"
echo "  good fix: $GOODFIX (strengthens the boundary test)"
echo "  bad fix:  $BADFIX (does not build)"

echo
echo "== gate mechanics: verify-catch on the good fix =="
echo "   (the same mutation-vs-spec/test-sensitivity machinery behind G3/G5 —"
echo "    a real confirmed catch, never a self-reported pass)"
"$WORK/packets" verify-catch -repo "$DEMO" -base "$BASE" -fix "$GOODFIX" -file adult.go -line 4

echo
echo "== gate mechanics: verify-catch on the bad fix =="
echo "   (an honest non-fabricated signal — never a fake pass on a broken revision)"
"$WORK/packets" verify-catch -repo "$DEMO" -base "$BASE" -fix "$BADFIX" -file adult.go -line 4

echo
echo "== adversarial inspection: packets probe =="
echo "   (seeds a known-bad revision in its OWN throwaway repo and runs it"
echo "    through the real gates — must report caught, never escaped)"
"$WORK/packets" probe

echo
echo "== booting the real server on the scratch repo =="
LEDGER="$WORK/ledger"
"$WORK/packets" -repo "$DEMO" -base "$BASE" -fix "$GOODFIX" -file adult.go -line 4 \
  -addr :3099 -ledger "$LEDGER" -bandwidth 5 \
  -backlog "base=$BASE,fix=$BADFIX,file=adult.go,line=4" \
  > "$WORK/server.log" 2>&1 &
SERVER_PID=$!
trap 'kill "$SERVER_PID" 2>/dev/null; rm -rf "$WORK"' EXIT
sleep 1

cat <<EOF

== the server is live on http://localhost:3099 ==

Everything past this point is a real UI action — open the URL above and:

  1. compose  — Console (/) shows the good-fix packet composing on load (it
                runs the gauntlet from -base/-fix immediately) AND a queued
                target for the bad fix under "queued targets:". Click its
                "compose" affordance to dispatch it too.
  2. forward  — the good-fix packet is a real confirmed catch: watch it
                settle into the "verified" rail as its gates pass.
  3. hold     — the bad-fix packet's build gate (G4) fails for real (it does
                not compile) — watch it land in "needs you" as a BLOCKING
                hold, not a fabricated verdict.
  4. inspect  — open the Inspector for either packet (click through, or go
                straight to http://localhost:3099/review?wo=1) to see its
                real diff, gate timeline, and (for the held one) the exact
                failing gate's detail.
  5. deliver  — once the good-fix packet is verified, ACK it as delivered
                from another terminal:
                  $WORK/packets deployed -ledger "$LEDGER" -session default -wo <its id>
                (find its id in the Console or Inspector url). A packet
                never renders delivered until this real, explicit ACK runs.

Press Ctrl-C to stop the server and clean up the scratch repo.
EOF

wait "$SERVER_PID"
