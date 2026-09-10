package cli

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"

	"github.com/joaomdsg/packets/internal/app"
	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// deployVerdict is the pure decision at the heart of `packets deployed`/
// `packets regressed`: what status (if any) to append, given whether a
// --check command was run and its result. With no check, the operator
// typing the command IS the evidence — it always proceeds. With a check,
// the check's exit code must AGREE with the asserted verb; a disagreement
// refuses rather than appending a status that contradicts what was just
// observed.
func deployVerdict(verb string, checkRan bool, checkErr error) (status, refusal string) {
	if !checkRan {
		return verb, ""
	}
	confirms := checkErr == nil // the check command exited 0
	switch verb {
	case "deployed":
		if !confirms {
			return "", fmt.Sprintf("check failed (%v) — not marking deployed", checkErr)
		}
	case "regressed":
		if confirms {
			return "", "check still passes — not marking regressed"
		}
	}
	return verb, ""
}

// runCheck runs the operator-supplied check command and reports whether it
// exited 0 — the "re-checkable" evidence a deploy/regression assertion can
// lean on, distinct from the assertion itself.
func runCheck(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return nil
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, out.String())
	}
	return nil
}

// runDeploy is the `deployed`/`regressed` subcommand: a host-issued ACK (or
// regression) for one packet, backed by an OPTIONAL re-checkable command
// whose exit code the host captures — never an agent's self-report. It
// reopens the existing ledger at -ledger (the same durable JetStream store
// -live/-backlog seed at boot) and appends the resulting status.
func runDeploy(verb string, args []string, out io.Writer) error {
	fs := flag.NewFlagSet(verb, flag.ContinueOnError)
	ledgerPath := fs.String("ledger", "catches", "durable economy store base (matches the running server's -ledger)")
	session := fs.String("session", "default", "session key the packet belongs to")
	wo := fs.Int("wo", 0, "the packet id to mark")
	check := fs.String("check", "", "an optional command whose exit code must agree with this verb (run via a shell)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *wo <= 0 {
		return fmt.Errorf("%s: -wo is required", verb)
	}

	ctx := context.Background()
	var checkErr error
	checkRan := *check != ""
	if checkRan {
		checkErr = runCheck(ctx, []string{"sh", "-c", *check})
	}
	status, refusal := deployVerdict(verb, checkRan, checkErr)
	if refusal != "" {
		return fmt.Errorf("%s: %s", verb, refusal)
	}

	f, err := fabric.Start(ctx, *ledgerPath+"-fabric")
	if err != nil {
		return fmt.Errorf("%s: open ledger: %w", verb, err)
	}
	defer f.Close()
	log := ledger.Bind(f, *session, app.LedgerInstance)
	if err := log.AppendStatus(*wo, status); err != nil {
		return fmt.Errorf("%s: %w", verb, err)
	}
	fmt.Fprintf(out, "pkt#%d: %s\n", *wo, status)
	return nil
}
