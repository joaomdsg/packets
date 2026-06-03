// Package review models the PR-review surface: anchored comment threads
// the reviewer acts on. Its first job is turning mutation-oracle findings
// into the `question:` threads a reviewer sees pinned to the diff.
package review

import "github.com/joaomdsg/agntpr/internal/mutation"

// Status is the lifecycle state of a review Thread.
type Status string

// Open is a thread awaiting the reviewer's attention.
const Open Status = "open"

// Thread is an anchored review comment (DESIGN §5): a Conventional
// Comment pinned to a line range in a file.
type Thread struct {
	File      string
	StartLine int
	EndLine   int
	Tag       string // Conventional Comment label, e.g. "question"
	Author    string // "agntpr" for oracle-authored threads
	Body      string
	Status    Status
}

// Render formats the thread as a Conventional Comment line: "<tag>: <body>".
func (t Thread) Render() string {
	return t.Tag + ": " + t.Body
}

// QuestionThreadsFromMutations turns each surviving-mutant finding into an
// open `question:` thread authored by the agent, anchored to the finding's
// line. Order is preserved.
func QuestionThreadsFromMutations(findings []mutation.Finding) []Thread {
	threads := make([]Thread, 0, len(findings))
	for _, f := range findings {
		threads = append(threads, Thread{
			File:      f.File,
			StartLine: f.Line,
			EndLine:   f.Line,
			Tag:       "question",
			Author:    "agntpr",
			Body:      f.Message,
			Status:    Open,
		})
	}
	return threads
}
