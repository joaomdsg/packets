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

/* ---- the setup surface (the Anthropic key) ---- */
.settings { display: flex; flex-direction: column; gap: var(--pk-md); }
/* configured/unconfigured are honest STATES, colored in the calm palette — never
   an alarm red/green. Unconfigured is dim (a calm "not yet"), configured reads in
   the balance hue (a live capability), mirroring the per-state convention. */
.settings__status[data-state="unconfigured"] { color: var(--pk-ink-dim); }
.settings__status[data-state="configured"] { color: var(--pk-balance); }
.settings__token { display: flex; gap: var(--pk-sm); align-items: baseline; }
.settings__token-input {
  padding: var(--pk-xs) var(--pk-sm);
  background: var(--pk-surface);
  color: var(--pk-ink);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  min-width: 22ch;
}
.settings__token-input:focus { outline: none; border-color: var(--pk-accent); }
.settings__save {
  padding: var(--pk-xs) var(--pk-md);
  background: var(--pk-surface-2);
  color: var(--pk-balance);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  cursor: pointer;
}
.settings__save:hover { border-color: var(--pk-balance); }

/* ---- the authoring assist (the producer's draft read) ---- */
.authoring { display: flex; flex-direction: column; gap: var(--pk-sm); }
.compose__analyze {
  padding: var(--pk-xs) var(--pk-md);
  background: var(--pk-surface-2);
  color: var(--pk-ink-dim);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  cursor: pointer;
}
.compose__analyze:hover { border-color: var(--pk-accent); color: var(--pk-ink); }
.analysis { display: flex; flex-direction: column; gap: var(--pk-xs); padding: var(--pk-sm) var(--pk-md); background: var(--pk-surface); border: 1px solid var(--pk-line); border-radius: 6px; }
.analysis__summary { color: var(--pk-ink); }
/* readiness is an honest STATE in the calm palette — never an alarm green/red. A
   blocked draft reads dim ("not yet"); a ready one reads in the balance hue. */
.analysis__readiness[data-state="blocked"] { color: var(--pk-ink-dim); }
.analysis__readiness[data-state="ready"] { color: var(--pk-balance); }
.analysis__questions-label { color: var(--pk-ink-dim); font-size: 0.92em; }
.analysis__questions { margin: 0; padding-left: var(--pk-md); color: var(--pk-ink); }
.analysis__unavailable { color: var(--pk-ink-dim); }
.authoring-editor { height: 160px; border: 1px solid var(--pk-line); border-radius: 6px; }
/* the flagged spans, by severity — a calm underline, never a red squiggle. */
.authoring-flag-question { text-decoration: underline dotted var(--pk-accent); }
.authoring-flag-gap { text-decoration: underline wavy var(--pk-ink-dim); }
.authoring-flag-note { text-decoration: underline dotted var(--pk-ink-dim); }
/* the live-read indicator: dim and hidden at rest, a calm "analyzing…" while a
   debounced re-read is pending/in-flight — never a spinner. */
.compose__analyzing { color: var(--pk-ink-dim); font-size: 0.85em; opacity: 0; transition: opacity 0.2s; }
.compose__analyzing[data-state="pending"], .compose__analyzing[data-state="analyzing"] { opacity: 1; }
/* the readiness reflection beside place — a guide, not an alarm: caution reads
   dim, ready reads in the balance hue. */
.compose__readiness { font-size: 0.9em; }
.compose__readiness[data-state="caution"] { color: var(--pk-ink-dim); }
.compose__readiness[data-state="ready"] { color: var(--pk-balance); }

/* ---- author a live order ---- */
.compose { display: flex; flex-direction: column; gap: var(--pk-sm); margin: var(--pk-sm) 0; }
.compose__prompt {
  padding: var(--pk-sm);
  background: var(--pk-surface);
  color: var(--pk-ink);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  min-height: 3.5em;
  resize: vertical;
}
.compose__prompt:focus { outline: none; border-color: var(--pk-accent); }
.compose__place {
  align-self: flex-start;
  padding: var(--pk-xs) var(--pk-md);
  background: var(--pk-surface-2);
  color: var(--pk-balance);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font: inherit;
  cursor: pointer;
}
.compose__place:hover { border-color: var(--pk-balance); }
.compose__needs-key { color: var(--pk-ink-dim); font-size: 0.92em; }
.compose__needs-key-link { color: var(--pk-accent); text-decoration: none; }
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
/* open review questions (surviving mutants) for a session — test debt the green
   verdict hides, a quiet accent link into that session's /review; only shown when
   there are any. Never an alarm. */
.board-row__questions { color: var(--pk-ink-dim); text-decoration: none; border-bottom: 1px dotted var(--pk-accent); }
.board-row__questions:hover { color: var(--pk-ink); }
/* a session's integration verdict, surfaced on the board only when it BLOCKS a
   merge — honest color (R45 palette): conflict = muted warn, checks-red = muted
   loss. Never an alarm. */
.board-row__land { font-size: 0.92em; }
.board-row__land[data-state="land-conflict"] { color: var(--pk-inflight); }
.board-row__land[data-state="land-checks-red"] { color: var(--pk-lost); }
/* fleet-level merge-readiness roll-up: how much of the fleet is blocked from
   landing. A calm dim summary line, surfaced only when ≥1 session is blocked — a
   count, never a gauge or alarm. */
.board__land-summary { display: block; color: var(--pk-ink-dim); font-size: 0.92em; }
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
/* a filled order's reviewable test-debt — how many open review questions it left;
   a quiet accent count (the dispatch→review tie), never an alarm. */
.board-row__dispatch-questions { color: var(--pk-ink-dim); }
/* "watch it fill": a calm live row while the runner fills an order — the cycle beats
   accruing as the oracle works. Dim mono, in the beat idiom; vanishes when done. */
.order-filling { color: var(--pk-ink-dim); font-family: var(--pk-mono); font-size: 0.92em; padding: var(--pk-xs) 0; }
/* the scrolling agent transcript while an order fills: bounded height so a long run
   scrolls in place rather than pushing the card; the calm mono idiom, no alarm. */
.order-transcript {
  margin: var(--pk-xs) 0;
  max-height: 14em;
  overflow-y: auto;
  padding: var(--pk-xs) var(--pk-sm);
  background: var(--pk-surface);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font-family: var(--pk-mono);
  font-size: 0.88em;
  color: var(--pk-ink-dim);
}
.order-transcript__line { padding: 1px 0; white-space: pre-wrap; word-break: break-word; }

/* ---- the single review card ---- */
.stock-row, .balance-row, .bandwidth-row, .dispatch-row, .beat-row, .review-card, .land-row, .onboarding {
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
.bandwidth-row__amount { margin: 0; color: var(--pk-accent); font-weight: 600; }
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
/* the Monaco review editor island: a sized mount point for the read-only editor.
   The editor is progressive enhancement over the text threads above; if it never
   mounts (loader blocked, JS off), this empty box just stays collapsed and the
   text threads carry the review. */
.review-editor-island { display: block; }
.review-editor { width: 100%; height: 60vh; border: 1px solid var(--pk-line); border-radius: 6px; }
/* the per-order diff editor — the edits the work order made, base vs fix side by
   side (a static, pre-funded diff; never a faked live agent). */
.order-diff-island { display: block; }
.order-diff-editor { width: 100%; height: 45vh; border: 1px solid var(--pk-line); border-radius: 6px; }
.review-editor:empty { height: 0; border: 0; } /* no editor mounted → no empty box */
/* the answer affordance: write a killing test + submit. Calm, in the surface idiom —
   a monospace input area + a quiet submit; the reward is the question vanishing, so
   nothing here shouts. */
.review-answer { display: flex; flex-direction: column; gap: var(--pk-xs); margin-top: var(--pk-sm); }
.review-answer__label { margin: 0; color: var(--pk-ink-dim); font-size: 0.92em; }
/* the editable Monaco answer pane: write the killing test in a real editor matching
   the read-only source pane above. */
.review-answer__input { display: flex; flex-direction: column; gap: var(--pk-xs); }
.review-answer__editor { width: 100%; height: 14em; border: 1px solid var(--pk-line); border-radius: 6px; }
.review-answer__submit {
  align-self: flex-start;
  color: var(--pk-ink); background: var(--pk-surface);
  border: 1px solid var(--pk-accent); border-radius: 4px;
  padding: 4px 12px; cursor: pointer;
}
/* the in-flight running status — calm dim text, shown by datastar (data-show) only
   while the oracle re-run is in flight. */
.review-answer__running { color: var(--pk-ink-dim); font-size: 0.92em; }
/* a surviving-mutant line in the editor: a calm left-edge accent + a glyph, never
   an alarm — the honest "the tests didn't catch this here" marker. */
.review-survivor-line { background: color-mix(in srgb, var(--pk-accent) 12%, transparent); }
.review-survivor-glyph { background: var(--pk-accent); width: 3px !important; margin-left: 2px; }
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

/* ---- the prep bench: the fundable work on deck, so the Lead curates what a Spend
   funds instead of a blind auto-pick. A calm mono list, no alarm. ---- */
.bench {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--pk-xs) var(--pk-sm);
  padding: var(--pk-xs) var(--pk-md);
  margin-bottom: var(--pk-sm);
  border-left: 2px solid var(--pk-line);
}
.bench__label { color: var(--pk-ink-dim); font-size: 0.82em; text-transform: uppercase; letter-spacing: 0.04em; }
/* each bench item is a fund-this button — a calm mono chip, balance-hue on hover
   (it spends a catch), never an alarm. */
.bench__item {
  padding: 1px var(--pk-sm);
  background: transparent;
  color: var(--pk-ink-dim);
  border: 1px solid var(--pk-line);
  border-radius: 6px;
  font-family: var(--pk-mono);
  font-size: 0.92em;
  cursor: pointer;
}
.bench__item:hover { color: var(--pk-balance); border-color: var(--pk-balance); }
`
