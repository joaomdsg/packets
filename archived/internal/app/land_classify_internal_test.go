package app

import "testing"

// The land control must tell the Lead which outcome they got — a clickable PR link on
// success, a guard notice when blocked, an error when the push failed — instead of one
// undifferentiated mono blob. classifyLandResult is the pure decision behind that: it
// reads the cached result string (whose shapes are fixed by setLandResult's call sites)
// into a kind, surfacing the URL only when a PR actually opened.
func TestClassifyLandResult_distinguishesEachOutcomeForTheLead(t *testing.T) {
	cases := []struct {
		name     string
		res      string
		wantKind landResultKind
		wantURL  string
	}{
		{"no result yet renders nothing", "", landResultNone, ""},
		{"an https PR URL is a clickable success", "https://github.com/o/r/pull/42", landResultOpened, "https://github.com/o/r/pull/42"},
		{"a PR URL whose path contains a keyword is still a success, not a guard match", "https://github.com/o/blocked/pull/9", landResultOpened, "https://github.com/o/blocked/pull/9"},
		{"a plain http PR URL is still a success", "http://example.test/pr/1", landResultOpened, "http://example.test/pr/1"},
		{"a guard message reads as blocked", "blocked — 2 open threads — resolve them first", landResultBlocked, ""},
		{"a push failure reads as an error", "forward failed — push rejected", landResultError, ""},
		{"an unrecognized non-empty string is treated as an error, never a clickable success", "weird unexpected value", landResultError, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, url := classifyLandResult(tc.res)
			if kind != tc.wantKind {
				t.Errorf("classifyLandResult(%q) kind = %v, want %v", tc.res, kind, tc.wantKind)
			}
			if url != tc.wantURL {
				t.Errorf("classifyLandResult(%q) url = %q, want %q", tc.res, url, tc.wantURL)
			}
		})
	}
}
