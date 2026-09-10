package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// reanchorAdjustment is the content-relocation core of the review-thread loop's
// "outdated / moved" signal (DESIGN §28, thin slice): after the agent settles a new
// revision, the Lead's adjustment must say whether the line they commented on is still
// there, moved, or was edited away — instead of silently pointing at a stale number.
func TestReanchorAdjustment_tellsTheLeadWhetherTheirLineSurvived(t *testing.T) {
	const file = "package main\n\nfunc add(a, b int) int {\n\treturn a + b\n}\n"
	cases := []struct {
		name      string
		origLine  string
		origNum   int
		content   string
		wantState adjAnchorState
		wantLine  int
	}{
		{"unchanged line at the same number is still anchored",
			"\treturn a + b", 4, file, adjSame, 4},
		{"the same content shifted down by inserted lines reads as moved",
			"\treturn a + b", 4, "// a new header comment\n// and another\n" + file, adjMoved, 6},
		{"the anchored line edited away is outdated — the agent addressed it",
			"\treturn a + b", 4, "package main\n\nfunc add(a, b int) int {\n\treturn a - b\n}\n", adjOutdated, 0},
		{"the anchored line deleted (file shorter) is outdated",
			"\treturn a + b", 4, "package main\n", adjOutdated, 0},
		{"content found elsewhere even when the old number is out of range reads as moved",
			"\treturn a + b", 999, file, adjMoved, 4},
		{"a \\r\\n file still matches a \\n-anchored line (trailing-CR tolerance)",
			"\treturn a + b", 4, "package main\r\n\r\nfunc add(a, b int) int {\r\n\treturn a + b\r\n}\r\n", adjSame, 4},
		{"an empty anchor has nothing to relocate and is outdated",
			"", 4, file, adjOutdated, 0},
		{"first match wins when a moved anchor's content appears more than once",
			"\tx := 1", 2, "\tx := 1\n\ty := 2\n\tx := 1\n", adjMoved, 1},
		{"an unchanged line at its own number stays Same, not reported moved to an earlier duplicate",
			"\tx := 1", 2, "\tx := 1\n\tx := 1\n", adjSame, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reanchorAdjustment(tc.origLine, tc.origNum, tc.content)
			if got.State != tc.wantState {
				t.Errorf("reanchorAdjustment(%q,%d,...) State = %v, want %v", tc.origLine, tc.origNum, got.State, tc.wantState)
			}
			if got.Line != tc.wantLine {
				t.Errorf("reanchorAdjustment(%q,%d,...) Line = %d, want %d", tc.origLine, tc.origNum, got.Line, tc.wantLine)
			}
		})
	}
}

// A real review leaves SEVERAL adjustments. relocateAdjustments relocates each anchor
// against its file's current content (an injected reader, no disk) and carries each
// anchor's own comment through — so the surface can show every adjustment's status, not
// just the last. Order is preserved.
func TestRelocateAdjustments_relocatesEachAnchorCarryingItsOwnComment(t *testing.T) {
	const same = "package p\n\nfunc f() bool {\n\treturn a >= b\n}\n"                      // "\treturn a >= b" at line 4
	const moved = "// added\n// added\npackage p\n\nfunc f() bool {\n\treturn a >= b\n}\n" // shifted to line 6
	const gone = "package p\n\nfunc f() bool {\n\treturn a > b\n}\n"                       // the line was edited away
	read := func(file string) string {
		switch file {
		case "same.go":
			return same
		case "moved.go":
			return moved
		case "gone.go":
			return gone
		}
		return ""
	}
	anchors := []adjAnchorRecord{
		{file: "same.go", line: 4, content: "\treturn a >= b", comment: "guard the nil case"},
		{file: "moved.go", line: 4, content: "\treturn a >= b", comment: "rename this var"},
		{file: "gone.go", line: 4, content: "\treturn a >= b", comment: "use >= not >"},
	}

	got := relocateAdjustments(anchors, read)
	require.Len(t, got, 3, "one status per adjustment, in order")

	assert.Equal(t, adjSame, got[0].State)
	assert.Equal(t, 4, got[0].Line)
	assert.Equal(t, "guard the nil case", got[0].Comment, "each status carries its own comment")
	assert.Equal(t, "same.go", got[0].File)

	assert.Equal(t, adjMoved, got[1].State)
	assert.Equal(t, 6, got[1].Line)
	assert.Equal(t, "rename this var", got[1].Comment)

	assert.Equal(t, adjOutdated, got[2].State)
	assert.Equal(t, "use >= not >", got[2].Comment)
}

// An anchor whose file the reader can't return (deleted / unreadable → "") relocates to
// Outdated (the line is gone as far as the surface can tell), never a panic.
func TestRelocateAdjustments_treatsAnUnreadableFileAsOutdated(t *testing.T) {
	got := relocateAdjustments(
		[]adjAnchorRecord{{file: "missing.go", line: 4, content: "\treturn a >= b", comment: "x"}},
		func(string) string { return "" },
	)
	require.Len(t, got, 1)
	assert.Equal(t, adjOutdated, got[0].State)
}

// No adjustments → no statuses (nothing to render).
func TestRelocateAdjustments_emptyForNoAnchors(t *testing.T) {
	assert.Empty(t, relocateAdjustments(nil, func(string) string { return "" }))
}

// Re-commenting the same file:line must REPLACE that adjustment (last-writer), not stack a
// duplicate badge — so the surface shows one entry per commented line with the latest
// comment. upsertAnchor replaces-in-place by file:line, else appends, preserving order.
func TestUpsertAnchor_replacesByFileAndLineElseAppends(t *testing.T) {
	rec := func(file string, line int, content, comment string) adjAnchorRecord {
		return adjAnchorRecord{file: file, line: line, content: content, comment: comment}
	}

	// Append to empty.
	got := upsertAnchor(nil, rec("a.go", 5, "x", "c1"))
	require.Equal(t, []adjAnchorRecord{rec("a.go", 5, "x", "c1")}, got)

	// Fresh base per case — upsert may replace in place, so each case must not see a
	// prior case's mutation.
	freshBase := func() []adjAnchorRecord {
		return []adjAnchorRecord{rec("a.go", 5, "x", "c1"), rec("b.go", 9, "y", "c2")}
	}

	// Same file:line → replaced IN PLACE (position preserved, content+comment updated).
	rep := upsertAnchor(freshBase(), rec("a.go", 5, "xx", "c1-new"))
	require.Len(t, rep, 2, "a re-comment on the same line does not grow the list")
	assert.Equal(t, rec("a.go", 5, "xx", "c1-new"), rep[0], "slot 0 updated to the latest content+comment")
	assert.Equal(t, rec("b.go", 9, "y", "c2"), rep[1], "the other anchor is untouched, order preserved")

	// Same file, DIFFERENT line → appended (distinct anchor).
	ap := upsertAnchor(freshBase(), rec("a.go", 6, "z", "c3"))
	require.Len(t, ap, 3)
	assert.Equal(t, rec("a.go", 6, "z", "c3"), ap[2])

	// Same line, DIFFERENT file → appended (distinct anchor).
	af := upsertAnchor(freshBase(), rec("c.go", 5, "w", "c4"))
	require.Len(t, af, 3)
	assert.Equal(t, rec("c.go", 5, "w", "c4"), af[2])
}

// The Lead must be able to RESOLVE (clear) an addressed adjustment so the list doesn't
// accumulate for the session. removeAnchor drops the entry matching file AND line (order
// preserved), and is a no-op for an unknown file:line. Twin of upsertAnchor.
func TestRemoveAnchor_dropsTheMatchingFileAndLineElseNoOp(t *testing.T) {
	rec := func(file string, line int) adjAnchorRecord {
		return adjAnchorRecord{file: file, line: line, content: "c", comment: "m"}
	}
	base := func() []adjAnchorRecord {
		return []adjAnchorRecord{rec("a.go", 5), rec("b.go", 9), rec("c.go", 12)}
	}

	// Remove the middle → order preserved, others untouched.
	got := removeAnchor(base(), "b.go", 9)
	require.Equal(t, []adjAnchorRecord{rec("a.go", 5), rec("c.go", 12)}, got)

	// Unknown file:line → unchanged.
	assert.Equal(t, base(), removeAnchor(base(), "nope.go", 1))

	// Remove from empty → empty.
	assert.Empty(t, removeAnchor(nil, "a.go", 5))

	// Same line, DIFFERENT file → NOT removed (both file AND line must match).
	assert.Equal(t, base(), removeAnchor(base(), "x.go", 5))
	// Same file, DIFFERENT line → NOT removed.
	assert.Equal(t, base(), removeAnchor(base(), "a.go", 6))
}

// jsStr produces a JS string literal safe to embed in a datastar expression: a file path
// containing a quote/backslash/newline must not break out of the quotes (injection guard
// for the resolve button's inline-assign).
func TestJsStr_escapesQuotesBackslashesNewlines(t *testing.T) {
	assert.Equal(t, `"plain.go"`, jsStr("plain.go"))
	assert.Equal(t, `"a\"b.go"`, jsStr(`a"b.go`), "a double-quote is escaped so it can't close the literal")
	assert.Equal(t, `"a\\b.go"`, jsStr(`a\b.go`), "a backslash is escaped")
	assert.Equal(t, `"a\nb"`, jsStr("a\nb"), "a newline can't break the expression onto a new line")
	assert.Equal(t, `"a\rb"`, jsStr("a\rb"))
	assert.Equal(t, `""`, jsStr(""))
}
