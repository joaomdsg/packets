package ledger

// Stock is the held quantity projected from the confirmed-catch ledger: how many
// real catches have been minted, tallied by reason and by the mint-time bits. It
// is a pure projection of the logged records — never a live counter, never an
// inferred number — so it can be audited against the JSONL at any time.
type Stock struct {
	Count            int
	ByReason         map[string]int
	SelfFlagged      int
	WouldHaveShipped int
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
