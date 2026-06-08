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
// derives that from reinvested/done. misses is done orders that minted nothing
// (done − reinvested, clamped at 0), mirroring BoardRows.
type fleetRow struct {
	Key        string `json:"key"`
	Balance    int    `json:"balance"`
	Confirmed  int    `json:"confirmed"`
	Reinvested int    `json:"reinvested"`
	Queued     int    `json:"queued"`
	Running    int    `json:"running"`
	Done       int    `json:"done"`
	Misses     int    `json:"misses"`
}

func encodeFleetFrame(fleet map[string]ledger.Projection) []byte {
	rows := make([]fleetRow, 0, len(fleet))
	for key, p := range fleet {
		stock := ledger.ConfirmedCatches(p.Records())
		counts := p.DispatchStatusCounts()
		misses := counts.Done - stock.Reinvested
		if misses < 0 {
			misses = 0
		}
		rows = append(rows, fleetRow{
			Key:        key,
			Balance:    p.Balance(),
			Confirmed:  stock.Count,
			Reinvested: stock.Reinvested,
			Queued:     counts.Queued,
			Running:    counts.Running,
			Done:       counts.Done,
			Misses:     misses,
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

		for fleet := range fleets {
			if _, err := w.Write(encodeFleetFrame(fleet)); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

// WatchFleet subscribes to the whole fabric's minted economy and emits a fresh
// per-session projection map (ledger.FleetProjection) on every committed event,
// across all sessions — history first (a late subscriber sees current state),
// then live. It is the cross-session board's stream-driven feed: the board
// reflects every session off the one stream, regardless of which producer wrote
// it.
//
// Like Watch, re-folding the whole fleet per event reuses the canonical fold,
// and canceling ctx is the only teardown — it stops the subscription and closes
// the channel, with a send guard so an abandoned consumer cannot leak the feeder
// goroutine. The caller MUST cancel ctx when done.
func WatchFleet(ctx context.Context, f *fabric.Fabric) (<-chan map[string]ledger.Projection, error) {
	events, err := f.Subscribe(ctx, fabric.FleetMintedSubject())
	if err != nil {
		return nil, err
	}

	out := make(chan map[string]ledger.Projection, 64)
	go func() {
		defer close(out)
		for range events {
			fleet, err := ledger.FleetProjection(ctx, f)
			if err != nil {
				return
			}
			select {
			case out <- fleet:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}
