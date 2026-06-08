package bridge

import (
	"encoding/json"
	"net/http"

	"github.com/joaomdsg/packets/internal/fabric"
	"github.com/joaomdsg/packets/internal/ledger"
)

// snapshot is the render-agnostic economy state carried by one SSE frame — the
// observable projection counts, with explicit JSON keys so the wire shape is
// stable regardless of the ledger's internal types.
type snapshot struct {
	Balance int `json:"balance"`
	Catches int `json:"catches"`
	Orders  int `json:"orders"`
	Queued  int `json:"queued"`
}

func encodeFrame(p ledger.Projection) []byte {
	// Marshaling a struct of ints cannot fail, so the error is discarded.
	b, _ := json.Marshal(snapshot{
		Balance: p.Balance(),
		Catches: len(p.Records()),
		Orders:  len(p.WorkOrders()),
		Queued:  len(p.QueuedWorkOrders()),
	})
	return append(append([]byte("data: "), b...), '\n', '\n')
}

// Handler serves one session's economy as an SSE stream: it sets the
// text/event-stream content type, then writes a JSON snapshot frame for every
// committed event (history first, then live) until the client disconnects. The
// request context is the teardown signal — when the browser goes away it cancels
// the underlying subscription, so no goroutine is left tailing a dead connection.
func Handler(f *fabric.Fabric, session, instance string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshots, err := Watch(r.Context(), f, session, instance)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		flusher, _ := w.(http.Flusher)
		if flusher != nil {
			flusher.Flush() // send headers immediately, before the first event
		}

		for p := range snapshots {
			if _, err := w.Write(encodeFrame(p)); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}
