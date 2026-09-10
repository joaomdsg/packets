package report

import (
	"fmt"
	"sort"
	"strings"
)

// FormatText renders r as a plain text table (§16: "no charts").
func FormatText(r *Report) string {
	var b strings.Builder

	fmt.Fprintln(&b, "GATE")
	fmt.Fprintf(&b, "  emit_reject:   %d\n", r.GateRejects)
	fmt.Fprintf(&b, "  emit_warn:     %d\n", r.GateWarns)
	fmt.Fprintf(&b, "  emit_override: %d\n", r.GateOverrides)
	if len(r.WarningCodeCounts) > 0 {
		fmt.Fprintln(&b, "  warning codes:")
		for _, code := range sortedKeys(r.WarningCodeCounts) {
			fmt.Fprintf(&b, "    %-40s %d\n", code, r.WarningCodeCounts[code])
		}
	}

	fmt.Fprintln(&b, "\nTERMINATION")
	fmt.Fprintf(&b, "  with human_approval:    %d\n", r.TerminateWithApproval)
	fmt.Fprintf(&b, "  without human_approval: %d\n", r.TerminateWithoutApproval)

	fmt.Fprintln(&b, "\nACCRETION")
	for _, cause := range sortedKeys(r.AccretionByCause) {
		fmt.Fprintf(&b, "  %-20s %d\n", cause, r.AccretionByCause[cause])
	}
	if len(r.CIFailureRepeats) > 0 {
		fmt.Fprintln(&b, "  ci_failure repeat failures:")
		for _, path := range sortedKeys(r.CIFailureRepeats) {
			fmt.Fprintf(&b, "    %-50s %d\n", path, r.CIFailureRepeats[path])
		}
	}

	fmt.Fprintln(&b, "\nAMEND")
	fmt.Fprintf(&b, "  total packets:             %d\n", r.PacketCount)
	fmt.Fprintf(&b, "  total amends:              %d\n", r.AmendCount)
	fmt.Fprintf(&b, "  mean amends/packet:        %.2f\n", r.AmendMeanPerPacket)
	fmt.Fprintf(&b, "  mean tokens to terminate:  %.1f\n", r.AmendMeanTokens)
	fmt.Fprintf(&b, "  mean minutes to terminate: %.1f\n", r.AmendMeanMinutes)

	fmt.Fprintln(&b, "\nHALTS")
	for _, reason := range sortedKeys(r.HaltsByReason) {
		fmt.Fprintf(&b, "  %-20s %d\n", reason, r.HaltsByReason[reason])
	}
	fmt.Fprintf(&b, "  budget_halt total:   %d\n", r.BudgetHalts)

	return b.String()
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
