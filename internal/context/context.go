package context

import (
	"fmt"
	"os"
	"strings"

	"github.com/joaomdsg/agntpr/internal/db"
)

type Comment struct {
	Author string
	Body   string
}

type PRComment struct {
	Author   string
	Body     string
	Path     string
	Line     int
	DiffHunk string
}

type Plan struct {
	Version  int
	Content  string
	Feedback string
	Approved bool
}

const ContextFileName = ".agntpr-context.md"

func Build(
	issue *db.Issue,
	comments []*Comment,
	prComments []*PRComment,
	plans []*Plan,
) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Issue #%d: %s\n\n", issue.Number, issue.Title))
	b.WriteString(issue.Body)
	b.WriteString("\n\n")

	if len(plans) > 0 {
		b.WriteString("## Plan History\n\n")
		for _, p := range plans {
			status := "❌ Rejected"
			if p.Approved {
				status = "✅ Approved"
			}
			b.WriteString(fmt.Sprintf("### Plan v%d (%s)\n\n", p.Version, status))
			b.WriteString(p.Content)
			b.WriteString("\n\n")
			if p.Feedback != "" {
				b.WriteString(fmt.Sprintf("**Feedback:** %s\n\n", p.Feedback))
			}
		}
	}

	if len(comments) > 0 {
		b.WriteString("## Comments\n\n")
		for _, c := range comments {
			b.WriteString(fmt.Sprintf("**@%s**: %s\n\n", c.Author, c.Body))
		}
	}

	if len(prComments) > 0 {
		b.WriteString("## Code Review Comments\n\n")
		for _, c := range prComments {
			b.WriteString(fmt.Sprintf("**@%s** on %s:%d\n", c.Author, c.Path, c.Line))
			b.WriteString(fmt.Sprintf("> %s\n\n", c.Body))
		}
	}

	return b.String()
}

func Save(workDir, content string) error {
	path := workDir + "/" + ContextFileName
	return os.WriteFile(path, []byte(content), 0644)
}
