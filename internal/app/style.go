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

/* ---- shared nav header (fleet ↔ session card) ---- */
.board-nav {
  display: flex;
  align-items: baseline;
  gap: var(--pk-md);
  padding-bottom: var(--pk-sm);
  margin-bottom: var(--pk-lg);
  border-bottom: 1px solid var(--pk-line);
}
.board-nav__home { color: var(--pk-ink); font-weight: 700; text-decoration: none; }
.board-nav__home:hover { color: var(--pk-accent); }
.board-nav__breadcrumb { display: inline-flex; align-items: baseline; gap: var(--pk-xs); color: var(--pk-ink-dim); font-size: 0.92em; }
.board-nav__crumb { color: var(--pk-ink-dim); text-decoration: none; }
.board-nav__crumb:hover { color: var(--pk-accent); }
.board-nav__sep { color: var(--pk-ink-dim); }
.board-nav__key { color: var(--pk-ink); }

/* ---- the fleet board ---- */
.board { display: flex; flex-direction: column; gap: var(--pk-sm); }
/* the fleet view's one command: create a session. A calm inline input + button,
   in the surface idiom — no modal, no alarm. */
.board-create { display: flex; gap: var(--pk-sm); align-items: baseline; margin-bottom: var(--pk-md); }
.board-create__key {
  padding: var(--pk-xs) var(--pk-sm);
  background: var(--pk-surface);
  color: var(--pk-ink);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
}
.board-create__key:focus { outline: none; border-color: var(--pk-accent); }
.board-create__btn {
  padding: var(--pk-xs) var(--pk-md);
  background: var(--pk-surface-2);
  color: var(--pk-balance);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  cursor: pointer;
}
.board-create__btn:hover { border-color: var(--pk-balance); }
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
.board-row__key { font-weight: 700; min-width: 7ch; color: inherit; text-decoration: none; }
.board-row__key:hover { color: var(--pk-accent); }
.board-row__stock { font-weight: 600; color: var(--pk-confirmed); }
.board-row__balance { color: var(--pk-balance); font-variant-numeric: tabular-nums; }
.board-row__activity { color: var(--pk-ink-dim); font-family: var(--pk-mono); font-size: 0.92em; }
.board-row__misses { color: var(--pk-lost); }
.board-row__hitrate { color: var(--pk-ink-dim); }
.board-row__backlog { color: var(--pk-ink-dim); }
/* retire a session from the fleet view — a quiet, low-emphasis control (dim until
   hover), never an alarm; only on non-default rows. */
.board-row__retire {
  margin-left: auto;
  padding: 0 var(--pk-sm);
  background: transparent;
  color: var(--pk-ink-dim);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  font-size: 0.85em;
  cursor: pointer;
}
.board-row__retire:hover { color: var(--pk-lost); border-color: var(--pk-lost); }

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
/* a resolved order's outcome, legible at a glance in the honest palette (extends
   the per-state color of R45 to the dispatch round-trip): caught is a calm
   confirmed, missed a muted loss — never an alarm red/green. A queued/running
   order has no data-outcome, so it stays neutral dim. */
.board-row__dispatch[data-outcome="caught"] { color: var(--pk-confirmed); }
.board-row__dispatch[data-outcome="missed"] { color: var(--pk-lost); }
/* the oracle's verdict for a resolved order — the WHY behind the outcome, shown as
   calm dim secondary detail (the outcome word already carries the color). */
.board-row__dispatch-why { color: var(--pk-ink-dim); }

/* ---- the single review card ---- */
.stock-row, .balance-row, .dispatch-row, .beat-row, .review-card, .land-row, .onboarding {
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
/* the gated open-question badge: a calm heads-up that the green verdict hides
   unkilled mutants — dim secondary text with a quiet accent edge, never an alarm.
   The full anchored threads live on the /review surface. */
.review-questions {
  display: block;
  padding: var(--pk-xs) var(--pk-md);
  margin-bottom: var(--pk-sm);
  border-left: 2px solid var(--pk-accent);
  color: var(--pk-ink-dim);
  font-size: 0.95em;
  text-decoration: none;
}
.review-questions:hover { color: var(--pk-ink); }

/* ---- the /review surface: the oracle's open "question:" threads ---- */
.review { display: flex; flex-direction: column; gap: var(--pk-sm); }
.review__lead { margin: 0 0 var(--pk-sm) 0; color: var(--pk-ink); font-weight: 600; }
.review__empty { color: var(--pk-ink-dim); padding: var(--pk-sm) var(--pk-md); }
.review-thread {
  display: flex;
  flex-direction: column;
  gap: var(--pk-xs);
  padding: var(--pk-sm) var(--pk-md);
  background: var(--pk-surface);
  border: 1px solid var(--pk-line);
  border-left: 2px solid var(--pk-accent);
  border-radius: 6px;
}
.review-thread__anchor { color: var(--pk-ink-dim); font-family: var(--pk-mono); font-size: 0.92em; }
.review-thread__body { color: var(--pk-ink); }
.land-row__headline { margin: 0 0 var(--pk-xs) 0; font-weight: 600; }
.land-row__detail { margin: 0; color: var(--pk-ink-dim); }

/* ---- per-state color: the verdict + integration the Lead reads, legible at a
   glance in the honest-state palette. Color REINFORCES the state the headline
   text already names (strip the CSS and the text still reads it); never an alarm
   red/green, never a gauge. ---- */
/* a real catch / a fully-tested ship-ready line — a thing that happened (calm confirmed) */
.review-card[data-state="catch"] .review-card__headline,
.review-card[data-state="tested"] .review-card__headline { color: var(--pk-confirmed); }
/* partial progress / oracle still running — pending, not done (working amber) */
.review-card[data-state="partial-catch"] .review-card__headline,
.review-card[data-state="in-flight"] .review-card__headline { color: var(--pk-inflight); }
/* the oracle ran and said nothing to catch / no mutable signal — neutral, not a loss */
.review-card[data-state="no-catch"] .review-card__headline,
.review-card[data-state="no-oracle-signal"] .review-card__headline { color: var(--pk-ink-dim); }
/* the anchor was lost (rename / edited) — the oracle couldn't follow (muted lost) */
.review-card[data-state="lost-via-rename"] .review-card__headline,
.review-card[data-state="anchor-edited"] .review-card__headline { color: var(--pk-lost); }
/* integration: clean (calm), conflict (muted warn, NOT alarm), checks-red (muted loss), pending (neutral) */
.land-row[data-state="land-clean"] .land-row__headline { color: var(--pk-confirmed); }
.land-row[data-state="land-conflict"] .land-row__headline { color: var(--pk-inflight); }
.land-row[data-state="land-checks-red"] .land-row__headline { color: var(--pk-lost); }
.land-row[data-state="land-pending"] .land-row__headline { color: var(--pk-ink-dim); }

/* ---- first-run onboarding affordance: shown only on a truly-fresh session
   (data-state="empty"). A calm guide to the core loop, not an alarm — a quiet
   accent rule, dim supporting text, no animation/gauge (guardrails). ---- */
.onboarding[data-state="empty"] {
  border-left: 2px solid var(--pk-accent);
}
.onboarding__lead { margin: 0 0 var(--pk-xs) 0; font-weight: 600; color: var(--pk-ink); }
.onboarding__step { margin: 0 0 var(--pk-xs) 0; color: var(--pk-ink-dim); font-size: 0.95em; }

/* ---- the Spend action: the Lead's core economic move, shown only when there is
   balance to spend. A calm, deliberate control in the balance hue — not an alarm,
   not a pulsing call-to-action. ---- */
.spend-action {
  margin: 0 0 var(--pk-sm) 0;
  padding: var(--pk-xs) var(--pk-md);
  background: var(--pk-surface-2);
  color: var(--pk-balance);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  cursor: pointer;
}
.spend-action:hover { border-color: var(--pk-balance); }
`
