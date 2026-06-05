package ledger

import "strings"

// Stock is the held quantity projected from the confirmed-catch ledger: how many
// real catches have been minted, tallied by reason and by the mint-time bits. It
// is a pure projection of the logged records — never a live counter, never an
// inferred number — so it can be audited against the JSONL at any time.
type Stock struct {
	Count            int
	ByReason         map[string]int
	SelfFlagged      int
	WouldHaveShipped int
	// Reinvested is the dispatch-minted share of Count — catches a SPEND bought by
	// dispatching distinct work (Producer "wo:<id>"), as opposed to a connect-cycle
	// mint. It is an ADDITIVE PARTITION of Count (connect-minted = Count − Reinvested),
	// so the surface can show compounding: a spend's catch is distinguishable from a
	// fresh mint, making the reinvestment chain legible rather than two equal bumps.
	Reinvested int
}

// ConfirmedCatches projects a Stock from a slice of records. It is a TOTAL
// function over the input: only records that are real catches (ShouldRecord)
// contribute, so a miswired non-catch record can never inflate the stock. No I/O,
// no in-memory counter — the count IS the records.
func ConfirmedCatches(recs []CatchRecord) Stock {
	s := Stock{ByReason: map[string]int{}}
	for _, r := range recs {
		if !ShouldRecord(r.Outcome) {
			continue
		}
		s.Count++
		if strings.HasPrefix(r.Producer, "wo:") {
			s.Reinvested++ // minted by a dispatched run — the spend-to-earn share
		}
		s.ByReason[r.ReasonTag]++
		if r.SelfFlagged {
			s.SelfFlagged++
		}
		if r.WouldHaveShipped {
			s.WouldHaveShipped++
		}
	}
	return s
}
