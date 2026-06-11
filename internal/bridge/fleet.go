package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// fleetRow is one session's line in a fleet SSE frame — the board's
// stream-derivable economy fields tagged with the session key. It omits the
// in-process-only BacklogRemaining, and carries no hit-rate string: a renderer
// derives that from caught/done. caught is the exact first-pass-hit count (done
// orders whose own run minted a wo:<id> catch — ledger.ScoutingReport); misses is
// the rest of the done orders (done − caught), mirroring BoardRows.
type fleetRow struct {
	Key        string `json:"key"`
	Balance    int    `json:"balance"`
	Confirmed  int    `json:"confirmed"`
	Reinvested int    `json:"reinvested"`
	Queued     int    `json:"queued"`
	Running    int    `json:"running"`
	Done       int    `json:"done"`
	Caught     int    `json:"caught"`
	Misses     int    `json:"misses"`
	InFlight   int    `json:"in_flight"` // producers' pending bets — never folded into confirmed (two-scores)
	Rejected   int    `json:"rejected"`  // verified-losses: bets the host verified and found no catch
}

func encodeFleetFrame(fleet map[string]ledger.FleetView) []byte {
	rows := make([]fleetRow, 0, len(fleet))
	for key, v := range fleet {
		stock := ledger.ConfirmedCatches(v.Records())
		counts := v.DispatchStatusCounts()
		// First-pass hits: the EXACT count of done orders whose own run minted a catch
		// (ScoutingReport gates a hit on the SAME order being done, so a catch on a
		// still-running order can't be misattributed). Caught ≤ Done by construction,
		// so misses = done − caught needs no clamp — mirrors BoardRows.
		sr := v.ScoutingReport()
		rows = append(rows, fleetRow{
			Key:        key,
			Balance:    v.Balance(),
			Confirmed:  stock.Count,
			Reinvested: stock.Reinvested,
			Queued:     counts.Queued,
			Running:    counts.Running,
			Done:       counts.Done,
			Caught:     sr.Caught,
			Misses:     counts.Done - sr.Caught,
			InFlight:   v.InFlight,
			Rejected:   v.Rejected,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Queued != rows[j].Queued {
			return rows[i].Queued > rows[j].Queued // most work awaiting drain first
		}
		// key-asc is the honest deterministic stream-side tie-break: the in-process
		// registration ordinal BoardRows uses is not on the stream.
		return rows[i].Key < rows[j].Key
	})
	// Marshaling a slice of int/string structs cannot fail, so the error is discarded.
	b, _ := json.Marshal(rows)
	return append(append([]byte("data: "), b...), '\n', '\n')
}

// FleetHandler serves the cross-session board as an SSE stream: it sets the
// text/event-stream content type, then writes one ordered JSON frame (an array
// of per-session rows) for every committed event across any session — history
// first, then live — until the client disconnects. The request context is the
// teardown signal, cancelling the underlying WatchFleet subscription so no
// goroutine tails a dead connection.
func FleetHandler(f *fabric.Fabric) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fleets, err := WatchFleet(r.Context(), f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush()
		}

		for board := range fleets {
			if _, err := w.Write(encodeFleetFrame(board)); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

// WatchFleet emits a fresh per-session board (ledger.FleetBoard) on every
// committed event across all sessions — history first (a late subscriber sees
// current state), then live. It wakes on the WHOLE event taxonomy
// (FleetEventsSubject: minted ∪ claim ∪ scratch), not only mints, so a producer's
// claim submission or the host's rejection verdict drives a live frame carrying
// the updated claim lifecycle (in-flight bets, verified-losses) — the board is
// never frozen until the next mint.
//
// Like Watch, re-folding the whole fleet per event reuses the canonical fold,
// and canceling ctx is the only teardown — it stops the subscription and closes
// the channel, with a send guard so an abandoned consumer cannot leak the feeder
// goroutine. The caller MUST cancel ctx when done.
//
// PERF (prototype-scale, accepted): widening the wake from minted-only to the
// whole taxonomy means EVERY event — including high-frequency discarded scratch
// fan-out — now drives a refold, and each refold is FleetBoard = TWO full
// ReplaySubject passes (minted + claim). So the cost is ~2 full-stream replays
// per committed event per connected viewer, up from 1-on-mint-only. At prototype
// scale (few sessions, few viewers, modest stream) this is fine; if the scratch
// rate or viewer count grows, debounce/coalesce wakes or fold incrementally
// rather than replaying the whole stream per event.
func WatchFleet(ctx context.Context, f *fabric.Fabric) (<-chan map[string]ledger.FleetView, error) {
	events, err := f.Subscribe(ctx, fabric.FleetEventsSubject())
	if err != nil {
		return nil, err
	}

	out := make(chan map[string]ledger.FleetView, 64)
	go func() {
		defer close(out)
		for range events {
			board, err := ledger.FleetBoard(ctx, f)
			if err != nil {
				return
			}
			select {
			case out <- board:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
