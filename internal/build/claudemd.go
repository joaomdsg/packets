package build

import (
	"embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/joaomdsg/packets/internal/packet"
	"github.com/joaomdsg/packets/internal/state"
)

//go:embed assets/claude.md.tmpl
var assetsFS embed.FS

var claudeMDTemplate = template.Must(template.ParseFS(assetsFS, "assets/claude.md.tmpl"))

// checkApprovedCmd is what the literal terminal entry "human_approval"
// becomes wherever a predicate is actually executed (§3, §12 step 2).
const humanApproval = "human_approval"

func checkApprovedCmd(slug string) string {
	return fmt.Sprintf("packets check-approved %s", slug)
}

func substituteApproval(slug, cmd string) string {
	if cmd == humanApproval {
		return checkApprovedCmd(slug)
	}
	return cmd
}

// terminalItem is one terminal predicate as shown in CLAUDE.md: the
// substituted command plus whether it's flagged NEW.
type terminalItem struct {
	Cmd   string
	IsNew bool
}

// isGateSuggested reports whether cmd was added to terminal by the emit
// gate (state.GateSuggestedTerminals) — the harness's record of which
// terminal checks don't exist in the repo yet and must be created (§13.7
// rule 2), as opposed to constraints/terminal the human wrote by hand.
func isGateSuggested(st *state.State, cmd string) bool {
	for _, s := range st.GateSuggestedTerminals {
		if s == cmd {
			return true
		}
	}
	return false
}

type claudeMDData struct {
	Slug        string
	Version     int
	Attempt     int
	Goal        string
	Context     string
	Constraints []string
	Terminal    []terminalItem
	FailureMD   string
	HaltMD      string
}

// RenderClaudeMD fills the §13.7 template for one attempt, substituting
// human_approval in shown predicates and splicing in failureMD/haltMD when
// non-empty.
func RenderClaudeMD(pkt *packet.Packet, st *state.State, failureMD, haltMD string) (string, error) {
	ctx := strings.TrimSpace(pkt.Context)
	if ctx == "" {
		ctx = "(none)"
	}

	data := claudeMDData{
		Slug:      st.Slug,
		Version:   st.Version,
		Attempt:   st.Attempt,
		Goal:      pkt.Goal,
		Context:   ctx,
		FailureMD: strings.TrimSpace(failureMD),
		HaltMD:    strings.TrimSpace(haltMD),
	}
	for _, c := range pkt.Constraints {
		data.Constraints = append(data.Constraints, substituteApproval(st.Slug, c))
	}
	for _, t := range pkt.Terminal {
		data.Terminal = append(data.Terminal, terminalItem{
			Cmd:   substituteApproval(st.Slug, t),
			IsNew: isGateSuggested(st, t),
		})
	}

	var buf strings.Builder
	if err := claudeMDTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("build: render CLAUDE.md: %s", err)
	}
	return buf.String(), nil
}

// FormatFailure renders failure.md in the exact §13.8 format. It is used
// by the ci node (Phase 5); provided and tested now because build.go
// writes runs/<attempt> artifacts a red CI run needs to reference.
func FormatFailure(attempt int, kind, cmd string, exitCode int, tail, diffStat string) string {
	return fmt.Sprintf(
		"attempt: %d\nfailed: %s `%s`\nexit: %d\n--- last 200 lines ---\n%s\n--- diff since previous attempt ---\n%s\n",
		attempt, kind, cmd, exitCode, tail, diffStat)
}
