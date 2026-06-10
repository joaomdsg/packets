package app

import "github.com/go-via/via/h"

// styleHead is the one inline <style> attached to every rendered page's <head>
// (via App.AppendToHead in NewServer). It carries the packets visual language —
// a calm "control-room" base — as a single stylesheet over the class hooks the
// board and card markup already emit, so it changes no server markup.
func styleHead() h.H { return h.StyleEl(h.Raw(packetsStyle)) }

// packetsStyle is the base visual language (council R43): restrained dark
// surface, system-font type, a calm spacing rhythm, and color that REINFORCES
// honest state (confirmed / pending bet / verified-lost / missed / balance) —
// never an alarm, a gauge, a progress bar, or a fabricated rank. Meaning lives in
// structure + labels; color is calm reinforcement, so the page still reads with
// the stylesheet stripped. Design tokens are `--pk-*` custom properties so later
// slices (nav, menus, flows) inherit one palette.
const packetsStyle = `
:root {
  /* surface + ink */
  --pk-bg: #14171a;
  --pk-surface: #1b1f24;
  --pk-surface-2: #222831;
  --pk-ink: #e6e8eb;
  --pk-ink-dim: #9aa3ad;
  --pk-line: #2b323b;
  /* honest-state hues — muted, never alarm red/green */
  --pk-confirmed: #6fb59a;   /* a calm teal-green: a minted catch, a thing that happened */
  --pk-balance:   #7fa6c4;   /* cool blue-gray: a spendable resource, ready to act */
  --pk-inflight:  #c2a878;   /* muted amber/tan: a pending bet, under verification */
  --pk-lost:      #b08a8a;   /* desaturated mauve: a verified-loss / miss — acknowledged, not shamed */
  --pk-accent:    #d4a574;   /* warm bronze: focus / keyboard cue (later slices) */
  /* type */
  --pk-font: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  --pk-mono: ui-monospace, "SF Mono", Menlo, Monaco, "Cascadia Code", monospace;
  /* spacing rhythm */
  --pk-xs: 4px;
  --pk-sm: 8px;
  --pk-md: 14px;
  --pk-lg: 22px;
}

body {
  margin: 0;
  padding: var(--pk-lg);
  background: var(--pk-bg);
  color: var(--pk-ink);
  font-family: var(--pk-font);
  font-size: 14px;
  line-height: 1.5;
}

/* ---- the fleet board ---- */
.board { display: flex; flex-direction: column; gap: var(--pk-sm); }
.board-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--pk-sm) var(--pk-md);
  padding: var(--pk-sm) var(--pk-md);
  background: var(--pk-surface);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
}
.board-row:hover { background: var(--pk-surface-2); }           /* a calm cue for the future keyboard nav */
.board-row__key { font-weight: 700; min-width: 7ch; }
.board-row__stock { font-weight: 600; color: var(--pk-confirmed); }
.board-row__balance { color: var(--pk-balance); font-variant-numeric: tabular-nums; }
.board-row__activity { color: var(--pk-ink-dim); font-family: var(--pk-mono); font-size: 0.92em; }
.board-row__misses { color: var(--pk-lost); }
.board-row__hitrate { color: var(--pk-ink-dim); }
.board-row__backlog { color: var(--pk-ink-dim); }

/* the producers' bet lifecycle — one sealed cluster, distinct from confirmed stock */
.board-row__bets, .board-row__dispatches {
  display: inline-flex;
  align-items: baseline;
  gap: var(--pk-xs) var(--pk-sm);
  padding: 1px var(--pk-sm);
  border-left: 2px solid var(--pk-line);
}
.board-row__bets-label, .board-row__dispatches-label {
  color: var(--pk-ink-dim);
  font-size: 0.82em;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.board-row__inflight { color: var(--pk-inflight); }
.board-row__rejected { color: var(--pk-lost); }
.board-row__dispatch { color: var(--pk-ink-dim); font-family: var(--pk-mono); font-size: 0.92em; }

/* ---- the single review card ---- */
.stock-row, .balance-row, .dispatch-row, .beat-row, .review-card, .land-row {
  padding: var(--pk-sm) var(--pk-md);
  background: var(--pk-surface);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  margin-bottom: var(--pk-sm);
}
.stock__count { font-weight: 700; color: var(--pk-confirmed); }
.stock__reinvested { color: var(--pk-confirmed); }
.stock__reason, .stock__self-flagged, .stock__would-ship { color: var(--pk-ink-dim); font-size: 0.92em; }
.balance-row__amount { margin: 0; color: var(--pk-balance); font-weight: 600; }
.dispatch-row__counts { margin: 0; color: var(--pk-ink-dim); font-family: var(--pk-mono); }
.beat { color: var(--pk-ink-dim); font-family: var(--pk-mono); }
.review-card__headline { margin: 0 0 var(--pk-xs) 0; font-weight: 600; }
.review-card__detail { margin: 0; color: var(--pk-ink-dim); }
.land-row__headline { margin: 0 0 var(--pk-xs) 0; font-weight: 600; }
.land-row__detail { margin: 0; color: var(--pk-ink-dim); }
`
