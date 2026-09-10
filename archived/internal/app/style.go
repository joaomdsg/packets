package app

import "github.com/go-via/via/h"

// styleHead is the one inline <style> attached to every rendered page's <head>
// (via App.AppendToHead in NewServer). It carries the packets visual language —
// a calm "control-room" base — as a single stylesheet over the class hooks the
// board and card markup already emit, so it changes no server markup.
func styleHead() h.H { return h.StyleEl(h.Raw(packetsStyle)) }

// packetsStyle is the base visual language: the design system's brand pack (council
// bootstrap 2026-07-04) ported verbatim from design/tokens/ — IBM Plex type,
// a dense operational spacing rhythm, and the state grammar (signal / verified
// / held / risk / agent / delivered) that IS the product. Color REINFORCES
// honest state — never an alarm, a gauge, a progress bar, or a fabricated
// rank. Meaning lives in structure + labels; the page still reads with the
// stylesheet stripped. Design tokens are named custom properties so later
// slices (nav, menus, flows) inherit one palette.
const packetsStyle = `
@import url("https://fonts.googleapis.com/css2?family=IBM+Plex+Sans:wght@400;500;600;700&family=IBM+Plex+Mono:wght@400;500;600;700&display=swap");

:root {
  /* grounds & surfaces (darkest -> lightest) */
  --ground: #0a0d13;
  --surface-deep: #080a0f;
  --surface-panel: #0f131b;
  --surface-card: #141924;
  --surface-raised: #1b212e;

  /* borders */
  --hairline: #1b212e;
  --border-faint: rgba(255,255,255,.08);
  --border-mid: #2a3242;
  --border-dashed: #3a4150;

  /* text ramp (brightest -> dimmest) */
  --ink: #e8ebf1;
  --text-body: #c3cad6;
  --text-muted: #98a1b2;
  --text-faint: #616b7c;
  --text-ghost: #4e5666;
  --text-disabled: #3a4150;

  /* the state grammar — these ARE the product */
  --signal: #4cc4d4;
  --on-signal: #08222a;
  --delivered: #2a7683;
  --delivered-mid: #357f8a;
  --verified: #46c08a;
  --held: #e6b23e;
  --risk: #f0666b;
  --risk-deep: #d1585c;
  --risk-muted: #cf9a97;
  --agent: #a78bfa;

  /* semantic aliases */
  --you: var(--signal);
  --accent: var(--signal);
  --add: var(--verified);
  --del: var(--risk);

  /* spacing (dense console rhythm) */
  --sp-1: 4px;
  --sp-2: 7px;
  --sp-3: 9px;
  --sp-4: 12px;
  --sp-5: 14px;
  --sp-6: 16px;
  --sp-7: 22px;
  --sp-8: 32px;

  /* radii */
  --r-glyph: 4px;
  --r-btn-sm: 7px;
  --r-card-sm: 8px;
  --r-ann: 9px;
  --r-card: 10px;
  --r-btn: 8px;
  --r-cta: 11px;
  --r-stat: 12px;
  --r-frame: 14px;
  --r-shell: 16px;
  --r-pill: 999px;

  /* component heights */
  --h-chip-sm: 15px;
  --h-chip: 17px;
  --h-chip-lg: 20px;
  --h-btn-sm: 26px;
  --h-btn: 28px;
  --h-btn-lg: 34px;
  --h-cta: 50px;

  /* type */
  --font-ui: 'IBM Plex Sans', system-ui, -apple-system, sans-serif;
  --font-mono: 'IBM Plex Mono', ui-monospace, "SF Mono", Menlo, monospace;

  /* dense operational scale (mono UI text) */
  --fs-micro: 9px;
  --fs-tiny: 9.5px;
  --fs-small: 10px;
  --fs-label: 10.5px;
  --fs-body-mono: 11px;
  --fs-body: 12.5px;
  --fs-emph: 13px;
  --fs-hero-stat: 36px;

  /* tracking */
  --track-kicker: .2em;
  --track-label: .1em;
  --track-chip: .05em;
  --track-word: .04em;

  --lh-ui: 1.45;
  --lh-prose: 1.7;
  --lh-dense: 1.55;

  /* effects — glows, shadows, washes */
  --glow-signal: 0 0 8px var(--signal);
  --glow-risk: 0 0 10px rgba(240,102,107,.55);
  --glow-btn: 0 6px 18px rgba(76,196,212,.24);
  --shadow-frame: 0 30px 70px rgba(0,0,0,.45);
  --shadow-tooltip: 0 12px 30px rgba(0,0,0,.5);
  --wash-signal: radial-gradient(90% 70% at 82% -12%, rgba(76,196,212,.08), transparent 55%);
}

/* motion — the three primitives the design allows: live pulse, held pulse
   (a sharper amber pulse for a blocking hold), and the settle flow sweep. */
@keyframes pk-pulse { 0%,100% { opacity:1 } 50% { opacity:.35 } }
@keyframes pk-held-pulse { 0%,100% { box-shadow:0 0 6px rgba(240,102,107,.35) } 50% { box-shadow:0 0 16px rgba(240,102,107,.75) } }
@keyframes pk-flow { 0% { transform:translateX(-140%) } 100% { transform:translateX(240%) } }

/* The WCAG 2.4.7 fix: a real, calm focus ring on every keyboard-focused
   control — the signal accent (the documented keyboard cue), an outline so it
   does not reflow. The per-component border-color swap stays as reinforcement. */
:focus-visible {
  outline: 2px solid var(--signal);
  outline-offset: 2px;
}

/* ---- the shared component layer: box CSS lives once here; each surface adds
   its semantic class via multi-class for hue/state/layout only ---- */
.pk-btn {
  padding: var(--sp-1) var(--sp-5);
  background: var(--surface-raised);
  color: var(--signal);
  border: 1px solid var(--hairline);
  border-radius: var(--r-btn);
  font: inherit;
  cursor: pointer;
}
.pk-btn:hover { border-color: var(--signal); }
.pk-btn:disabled { color: var(--text-muted); cursor: default; }
.pk-btn:disabled:hover { border-color: var(--hairline); }
.pk-btn--quiet { background: transparent; color: var(--text-muted); }
.pk-btn--quiet:hover { border-color: var(--signal); color: var(--ink); }
.pk-input {
  padding: var(--sp-1) var(--sp-3);
  background: var(--surface-panel);
  color: var(--ink);
  border: 1px solid var(--hairline);
  border-radius: var(--r-btn);
  font: inherit;
}
.pk-input:focus-visible { border-color: var(--signal); }
.pk-chip {
  padding: 1px var(--sp-3);
  border: 1px solid var(--hairline);
  border-radius: var(--r-btn);
  font-family: var(--font-mono);
  font-size: var(--fs-small);
  color: var(--text-muted);
}
.pk-section-label {
  color: var(--text-faint);
  font-size: var(--fs-micro);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.pk-card {
  padding: var(--sp-3) var(--sp-5);
  background: var(--surface-panel);
  border: 1px solid var(--hairline);
  border-radius: var(--r-btn);
}

body {
  margin: 0;
  padding: var(--sp-7);
  background: var(--ground);
  color: var(--ink);
  font-family: var(--font-ui);
  font-size: 14px;
  line-height: 1.5;
}

/* ---- shared nav header (fleet ↔ session card) ---- */
.board-nav {
  display: flex;
  align-items: baseline;
  gap: var(--sp-5);
  padding-bottom: var(--sp-3);
  margin-bottom: var(--sp-7);
  border-bottom: 1px solid var(--hairline);
}
.board-nav__home { color: var(--ink); font-weight: 700; text-decoration: none; }
.board-nav__home:hover { color: var(--signal); }
.board-nav__breadcrumb { display: inline-flex; align-items: baseline; gap: var(--sp-1); color: var(--text-muted); font-size: var(--fs-small); }
.board-nav__crumb { color: var(--text-muted); text-decoration: none; }
.board-nav__crumb:hover { color: var(--signal); }
.board-nav__sep { color: var(--text-muted); }
.board-nav__key { color: var(--ink); }

/* ---- the packets brand mark + lockup (the design system's brand pack, locked) ----
   Every dimension derives from the one --mark-cell custom property the Go
   helper sets inline, via calc() — so packetMark(cell) stays a pure function
   of one parameter, never a fixed-size CSS class per call site. */
.pk-mark {
  display: inline-grid;
  grid-template-columns: repeat(2, var(--mark-cell));
  grid-auto-rows: var(--mark-cell);
  gap: max(1.5px, calc(var(--mark-cell) * 0.27));
  flex: none;
}
.pk-mark__cell {
  display: inline-block;
  width: var(--mark-cell);
  height: var(--mark-cell);
  border-radius: max(1.5px, calc(var(--mark-cell) * 0.23));
  box-sizing: border-box;
}
.pk-mark__cell--signal { background: var(--signal); }
.pk-mark__cell--delivered { background: var(--delivered); }
/* the small-size TR fallback (cell < 14px): the ghost's stroke would go
   sub-pixel, so it reads as a solid silhouette instead (locked rule). */
.pk-mark__cell--delivered-mid { background: var(--delivered-mid); }
/* the ghost "composing" TR cell (cell >= 14px only) — same edge, same addr,
   that fills solid --delivered once forwarded. */
.pk-mark__cell--ghost {
  background: color-mix(in srgb, var(--delivered) 8%, transparent);
  border: max(2px, calc(var(--mark-cell) * 0.18)) solid var(--delivered);
  transform: scale(1.035);
}
.pk-mark__cell--held { background: var(--risk); animation: pk-held-pulse 2s ease-in-out infinite; }

/* the compact in-chrome lockup: mark + stacked "packets / LABEL" — the only
   wordmark form allowed beside a breadcrumb (never the full inline wordmark). */
.pk-lockup { display: inline-flex; align-items: center; gap: calc(var(--mark-cell) * 1.2); }
.pk-lockup__labels { display: flex; flex-direction: column; gap: 2px; }
.pk-lockup__word {
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: calc(var(--mark-cell) * 1.7);
  letter-spacing: var(--track-word);
  color: var(--ink);
  line-height: 1;
  text-transform: lowercase;
}
.pk-lockup__sub {
  font-family: var(--font-mono);
  font-weight: 700;
  font-size: var(--fs-micro);
  letter-spacing: .16em;
  color: var(--text-faint);
  line-height: 1;
  text-transform: uppercase;
}

/* ---- the fleet board ---- */
.board { display: flex; flex-direction: column; gap: var(--sp-3); }
/* the fleet view's one command: create a session. A calm inline input + button,
   in the surface idiom — no modal, no alarm. */
.board-create { display: flex; flex-wrap: wrap; gap: var(--sp-3); align-items: baseline; margin-bottom: var(--sp-5); }
/* box CSS now lives on .pk-input / .pk-btn (multi-class); the semantic classes
   are kept as stable hooks for future per-surface nudges. */
/* the repo pick: the chosen full path shows quietly in mono, beside the Browse
   control that opens the server-side directory picker. */
.board-create__repo { display: flex; gap: var(--sp-3); align-items: baseline; }
.board-create__selected { font-family: var(--font-mono); font-size: var(--fs-micro); color: var(--text-muted); overflow-wrap: anywhere; }
/* the server-side directory picker: a calm surface panel that drops to its own
   full-width row below the create controls (so navigating the filesystem never
   crowds them), scrollable when a directory is deep. */
.board-create__browser {
  flex-basis: 100%;
  display: flex; flex-direction: column; gap: var(--sp-1);
  margin-top: var(--sp-3); padding: var(--sp-3);
  background: var(--surface-raised); border: 1px solid var(--hairline); border-radius: var(--r-btn);
  max-height: 320px; overflow-y: auto;
}
.board-create__browser-head { display: flex; gap: var(--sp-3); align-items: center; margin-bottom: var(--sp-1); }
.board-create__browser-dir { flex: 1; font-family: var(--font-mono); font-size: var(--fs-micro); color: var(--text-muted); overflow-wrap: anywhere; }
/* the navigable rungs — up + child folders — read as a calm left-aligned list
   (borderless, mono), the accent cueing the hover target, not a grid of buttons. */
.board-create__browser-up,
.board-create__browser-entry {
  text-align: left; background: transparent; border: 0; cursor: pointer;
  padding: var(--sp-1) var(--sp-3); border-radius: var(--r-glyph);
  font-family: var(--font-mono); font-size: var(--fs-micro); color: var(--ink);
}
.board-create__browser-up { color: var(--text-muted); }
.board-create__browser-up:hover,
.board-create__browser-entry:hover { background: var(--surface-panel); color: var(--signal); }

/* ---- the setup surface (the Anthropic key) ---- */
.settings { display: flex; flex-direction: column; gap: var(--sp-5); }
/* configured/unconfigured are honest STATES, colored in the calm palette — never
   an alarm red/green. Unconfigured is dim (a calm "not yet"), configured reads in
   the balance hue (a live capability), mirroring the per-state convention. */
.settings__status[data-state="unconfigured"] { color: var(--text-muted); }
.settings__status[data-state="configured"] { color: var(--signal); }
.settings__token { display: flex; gap: var(--sp-3); align-items: baseline; }
.settings__token-input { min-width: 22ch; }  /* box CSS on .pk-input */
/* .settings__save box CSS now on .pk-btn */

/* ---- the authoring assist (the assist's draft read) ---- */
.authoring { display: flex; flex-direction: column; gap: var(--sp-3); }
/* .compose__analyze is a .pk-btn--quiet; it kept the raised-surface background */
.compose__analyze { background: var(--surface-raised); }
.analysis { display: flex; flex-direction: column; gap: var(--sp-1); }  /* box CSS on .pk-card */
.analysis__summary { color: var(--ink); }
/* readiness is an honest STATE in the calm palette — never an alarm green/red. A
   blocked draft reads dim ("not yet"); a ready one reads in the balance hue. */
.analysis__readiness[data-state="blocked"] { color: var(--text-muted); }
.analysis__readiness[data-state="ready"] { color: var(--signal); }
.analysis__flags-label { color: var(--text-muted); font-size: var(--fs-small); }
.analysis__flag { display: flex; flex-direction: column; gap: var(--sp-1); padding: var(--sp-2); }
.analysis__flag--nav { cursor: pointer; }
.analysis__flag--nav:hover { border-color: color-mix(in srgb, var(--signal) 40%, var(--hairline)); }
.analysis__flag-severity { align-self: flex-start; font-size: var(--fs-micro); font-weight: 700; text-transform: uppercase; letter-spacing: .04em; color: var(--held); }
.analysis__flag-head { display: flex; align-items: center; gap: var(--sp-2); }
.analysis__flag-where { margin-left: auto; color: var(--text-faint); font-size: var(--fs-micro); }
.analysis__flag-note { color: var(--ink); font-size: var(--fs-small); }
.analysis__questions-label { color: var(--text-muted); font-size: var(--fs-small); }
/* each question is now an answerable form block (text + choices + a note), so the
   list drops its bullets and stacks the questions with a calm rhythm. */
.analysis__questions { margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: var(--sp-3); color: var(--ink); }
.analysis__question { display: flex; flex-direction: column; gap: var(--sp-1); }
.analysis__question-text { color: var(--ink); }
/* the suggested answers: a wrap of quiet pickable chips (radios for pick-one,
   checkboxes for pick-any) — the native input stays, the label gives it a hit area. */
.analysis__choices { display: flex; flex-wrap: wrap; gap: var(--sp-3); }
.analysis__choice { display: inline-flex; align-items: center; gap: var(--sp-1); padding: var(--sp-1) var(--sp-3); background: var(--surface-raised); border: 1px solid var(--hairline); border-radius: var(--r-glyph); font-size: var(--fs-small); cursor: pointer; }
.analysis__choice:hover { border-color: var(--signal); }
/* the free-text note / different-answer input, and the single update control. */
.analysis__note { background: var(--surface-raised); border: 1px solid var(--hairline); border-radius: var(--r-glyph); color: var(--ink); padding: var(--sp-1) var(--sp-3); font-size: var(--fs-small); }
.analysis__note:focus { outline: none; border-color: var(--signal); }
.analysis__update { align-self: flex-start; margin-top: var(--sp-1); }
.analysis__unavailable { color: var(--text-muted); }
/* the editable Monaco editor — the single draft source (was a plain textarea). */
.compose__live { display: flex; flex-direction: column; gap: var(--sp-3); }
.compose__editor { height: 180px; border: 1px solid var(--hairline); border-radius: var(--r-btn); }
/* the flagged spans, by severity — a calm underline, never a red squiggle. */
.authoring-flag-question { text-decoration: underline dotted var(--signal); }
.authoring-flag-gap { text-decoration: underline wavy var(--text-muted); }
.authoring-flag-note { text-decoration: underline dotted var(--text-muted); }
/* the live-read indicator: dim and hidden at rest, a calm "analyzing…" while a
   debounced re-read is pending/in-flight — never a spinner. */
.compose__analyzing { color: var(--text-muted); font-size: var(--fs-micro); opacity: 0; transition: opacity 0.2s; }
.compose__analyzing[data-state="pending"], .compose__analyzing[data-state="analyzing"] { opacity: 1; }
/* the readiness reflection beside place — a guide, not an alarm: caution reads
   dim, ready reads in the balance hue. */
.compose__readiness { font-size: var(--fs-small); }
.compose__readiness[data-state="caution"] { color: var(--text-muted); }
.compose__readiness[data-state="ready"] { color: var(--signal); }

/* ---- author a live packet ---- */
.compose { display: flex; flex-direction: column; gap: var(--sp-3); margin: var(--sp-3) 0; }
.compose__place { align-self: flex-start; }  /* box CSS on .pk-btn */
.compose__needs-key { color: var(--text-muted); font-size: var(--fs-small); }
.compose__needs-key-link { color: var(--signal); text-decoration: none; }
/* box CSS (padding + surface + border + radius) now on .pk-card (multi-class);
   this keeps only the row's flex layout. */
.board-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--sp-3) var(--sp-5);
}
.board-row:hover { background: var(--surface-raised); }           /* a calm cue for the future keyboard nav */
.board-row__key { font-weight: 700; min-width: 7ch; color: inherit; text-decoration: none; }
.board-row__key:hover { color: var(--signal); }
.board-row__stock { font-weight: 600; color: var(--verified); }
.board-row__balance { color: var(--signal); font-variant-numeric: tabular-nums; }
.board-row__activity { color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-small); }
.board-row__misses { color: var(--risk); }
.board-row__hitrate { color: var(--text-muted); }
.board-row__backlog { color: var(--text-muted); }
/* open review questions (surviving mutants) for a session — test debt the green
   verdict hides, a quiet accent link into that session's /review; only shown when
   there are any. Never an alarm. */
.board-row__questions { color: var(--text-muted); text-decoration: none; border-bottom: 1px dotted var(--signal); }
.board-row__questions:hover { color: var(--ink); }
/* a session's integration verdict, surfaced on the board only when it BLOCKS a
   merge — honest color (the honest state palette): conflict = muted warn, checks-red = muted
   loss. Never an alarm. */
.board-row__land { font-size: var(--fs-small); }
.board-row__land[data-state="land-conflict"] { color: var(--signal); }
.board-row__land[data-state="land-checks-red"] { color: var(--risk); }
/* post-open lifecycle across the fleet (§29.2: Landed ≠ Merged) — terminal outcomes only */
.board-row__lifecycle { font-size: var(--fs-small); }
.board-row__lifecycle[data-state="merged"] { color: var(--verified); }
.board-row__lifecycle[data-state="bounced"] { color: var(--risk); }
/* fleet-level merge-readiness roll-up: how much of the fleet is blocked from
   landing. A calm dim summary line, surfaced only when ≥1 session is blocked — a
   count, never a gauge or alarm. */
.board__land-summary { display: block; color: var(--text-muted); font-size: var(--fs-small); }
/* retire a session from the fleet view — a quiet, low-emphasis control (dim until
   hover), never an alarm; only on non-default rows. */
/* a quiet variant (.pk-btn--quiet); keeps its margin, tighter padding, smaller
   type, and the lost-hue hover as semantic reinforcement. */
.board-row__retire { margin-left: auto; padding: 0 var(--sp-3); font-size: var(--fs-micro); }
.board-row__retire:hover { color: var(--risk); border-color: var(--risk); }

/* the peers' bet lifecycle — one sealed cluster, distinct from confirmed stock */
.board-row__bets, .board-row__sends {
  display: inline-flex;
  align-items: baseline;
  gap: var(--sp-1) var(--sp-3);
  padding: 1px var(--sp-3);
  border-left: 2px solid var(--hairline);
}
/* the uppercase labels are .pk-section-label (multi-class) */
.board-row__inflight { color: var(--signal); }
.board-row__rejected { color: var(--risk); }
.board-row__send { color: var(--text-muted); }  /* font-family/font-size now on .pk-chip; KEEP color + the [data-outcome] hue rules */
/* a resolved packet's outcome, legible at a glance in the honest palette (extends
   the honest per-state palette to the send round-trip): caught is a calm
   confirmed, missed a muted loss — never an alarm red/green. A queued/running
   packet has no data-outcome, so it stays neutral dim. */
.board-row__send[data-outcome="caught"] { color: var(--verified); }
.board-row__send[data-outcome="missed"] { color: var(--risk); }
/* the oracle's verdict for a resolved packet — the WHY behind the outcome, shown as
   calm dim secondary detail (the outcome word already carries the color). */
.board-row__send-why { color: var(--text-muted); }
/* a filled packet's reviewable test-debt — how many open review questions it left;
   a quiet accent count (the send→review tie), never an alarm. */
.board-row__send-questions { color: var(--text-muted); }
/* a settled packet with no open questions still drills into its base→fix diff — so
   a clean fill is never a dead end. Same quiet accent as the question count. */
.board-row__send-inspect { color: var(--text-muted); }
/* "watch it fill": a calm live row while the runner fills a packet — the cycle beats
   accruing as the oracle works. Dim mono, in the beat idiom; vanishes when done. */
.packet-filling { color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-small); padding: var(--sp-1) 0; }
/* the scrolling agent transcript while a packet fills: bounded height so a long run
   scrolls in place rather than pushing the card; the calm mono idiom, no alarm. */
.packet-transcript {
  margin: var(--sp-1) 0;
  max-height: 14em;
  overflow-y: auto;
  padding: var(--sp-1) var(--sp-3);
  background: var(--surface-panel);
  border: 1px solid var(--hairline);
  border-radius: var(--r-btn);
  font-family: var(--font-mono);
  font-size: var(--fs-small);
  color: var(--text-muted);
}
.packet-transcript__line { padding: 1px 0; white-space: pre-wrap; word-break: break-word; }

/* ---- the single review card ---- */
/* box CSS (padding + surface + border + radius) now on .pk-card (multi-class);
   these keep only the shared bottom margin between stacked cards. The
   stock/balance/bandwidth/send meter rows are RETIRED from the UI
   (the vocabulary map) — their render helpers are gone,
   so no rule for them lives here anymore. */
.beat-row, .review-card, .land-row, .onboarding {
  margin-bottom: var(--sp-3);
}
.beat { color: var(--text-muted); font-family: var(--font-mono); }
.review-card__headline { margin: 0 0 var(--sp-1) 0; font-weight: 600; }
.review-card__detail { margin: 0; color: var(--text-muted); }
/* the gated open-question badge: a calm heads-up that the green verdict hides
   unkilled mutants — dim secondary text with a quiet accent edge, never an alarm.
   The full anchored threads live on the /review surface. */
.review-questions {
  display: block;
  padding: var(--sp-1) var(--sp-5);
  margin-bottom: var(--sp-3);
  border-left: 2px solid var(--signal);
  color: var(--text-muted);
  font-size: var(--fs-small);
  text-decoration: none;
}
.review-questions:hover { color: var(--ink); }

/* ---- the /review surface: the oracle's open "question:" threads ---- */
.review { display: flex; flex-direction: column; gap: var(--sp-3); }
.review__lead { margin: 0 0 var(--sp-3) 0; color: var(--ink); font-weight: 600; }
.review__empty { color: var(--text-muted); padding: var(--sp-3) var(--sp-5); }
.review-thread {
  display: flex;
  flex-direction: column;
  gap: var(--sp-1);
  padding: var(--sp-3) var(--sp-5);
  background: var(--surface-panel);
  border: 1px solid var(--hairline);
  border-left: 2px solid var(--signal);
  border-radius: var(--r-btn);
}
.review-thread__anchor { color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-small); }
.review-thread__body { color: var(--ink); }
/* the Monaco review editor island: a sized mount point for the read-only editor.
   The editor is progressive enhancement over the text threads above; if it never
   mounts (loader blocked, JS off), this empty box just stays collapsed and the
   text threads carry the review. */
.review-editor-island { display: block; }
.review-editor { width: 100%; height: 60vh; border: 1px solid var(--hairline); border-radius: var(--r-btn); }
/* the per-packet diff editor — the edits the packet made, base vs fix side by
   side (a static, pre-funded diff; never a faked live agent). */
.packet-diff-island { display: block; }
.packet-diff-editor { width: 100%; height: 45vh; border: 1px solid var(--hairline); border-radius: var(--r-btn); }
/* the changed-file tree — the review surface's left rail: the full fix tree as
   native collapsible <details> groups, expanded by default. Changed leaves take
   the bronze accent, deletions the muted-mauve loss hue (both existing
   honest-state tokens — no new color), the open file a quiet raised background. */
.file-tree { display: flex; flex-direction: column; gap: 2px; font-family: var(--font-mono); font-size: var(--fs-small); }
.file-tree__summary { color: var(--text-muted); font-size: var(--fs-micro); padding: 0 var(--sp-1) var(--sp-1); margin-bottom: var(--sp-1); border-bottom: 1px solid var(--hairline); }
.file-tree__dir { cursor: pointer; color: var(--text-muted); padding: 2px 0; }
.file-tree__children { padding-left: var(--sp-3); border-left: 1px solid var(--hairline); margin-left: var(--sp-1); }
.file-tree__file { display: flex; align-items: baseline; gap: var(--sp-1); padding: 1px var(--sp-1); color: var(--ink); text-decoration: none; border-radius: var(--r-glyph); }
.file-tree__file:hover { background: var(--surface-raised); }
.file-tree__file--changed { color: var(--signal); }
.file-tree__file--deleted { color: var(--risk); text-decoration: line-through; }
.file-tree__file--selected { background: var(--surface-raised); outline: 1px solid var(--signal); }
.file-tree__counts { margin-left: auto; color: var(--text-muted); font-size: var(--fs-micro); }
.file-tree__badge { margin-left: var(--sp-1); min-width: 15px; height: 15px; padding: 0 4px; display: inline-flex; align-items: center; justify-content: center; border-radius: 8px; background: color-mix(in srgb, var(--signal) 16%, var(--surface-card)); color: var(--signal); font-size: var(--fs-micro); font-weight: 700; }
.review-editor:empty { height: 0; border: 0; } /* no editor mounted → no empty box */
/* the answer affordance: write a killing test + submit. Calm, in the surface idiom —
   a monospace input area + a quiet submit; the reward is the question vanishing, so
   nothing here shouts. */
.review-answer { display: flex; flex-direction: column; gap: var(--sp-1); margin-top: var(--sp-3); }
.review-answer__label { margin: 0; color: var(--text-muted); font-size: var(--fs-small); }
/* the editable Monaco answer pane: write the killing test in a real editor matching
   the read-only source pane above. */
.review-answer__input { display: flex; flex-direction: column; gap: var(--sp-1); }
.review-answer__editor { width: 100%; height: 14em; border: 1px solid var(--hairline); border-radius: var(--r-btn); }
/* reuses .pk-btn for the hairline/box; only reinforces the accent on the border
   and tightens the corner — .pk-btn owns the border width/style. */
.review-answer__submit {
  align-self: flex-start;
  color: var(--ink); background: var(--surface-panel);
  border-color: var(--signal); border-radius: var(--r-glyph);
  padding: 4px 12px; cursor: pointer;
}
/* the in-flight running status — calm dim text, shown by datastar (data-show) only
   while the oracle re-run is in flight. */
.review-answer__running { color: var(--text-muted); font-size: var(--fs-small); }
/* the adjustment entry point: leave an anchored comment and the live harness re-edits
   in place. A calm inline file/line/comment row in the surface idiom. */
.review-adjust { display: flex; flex-wrap: wrap; gap: var(--sp-3); align-items: baseline; margin-top: var(--sp-3); }
.review-adjust__label { flex-basis: 100%; margin: 0; color: var(--text-muted); font-size: var(--fs-small); }
.review-adjust__file { flex: 1 1 12em; }
.review-adjust__line { flex: 0 0 5em; }
.review-adjust__text { flex: 2 1 18em; }
/* the last adjustment's outcome after the agent settled a revision (DESIGN §28 thin
   slice): a calm "still here" notice, a confirmed-green "moved/addressed", or a dim
   "line edited" — the visible payoff of leaving an adjustment. */
.review-adjust__status { flex-basis: 100%; font-size: var(--fs-small); }
.review-adjust__status--same { color: var(--text-muted); }
.review-adjust__status--moved { color: var(--verified); }
.review-adjust__status--outdated { color: var(--text-muted); }
/* approve & open a PR: the land control. The result (PR URL / guard / failure) reads
   in mono so a URL is selectable and a guard message stands apart from the buttons. */
.land-control { display: flex; flex-wrap: wrap; gap: var(--sp-3); align-items: center; margin-top: var(--sp-3); }
.land-control__override { display: inline-flex; align-items: center; gap: var(--sp-1); color: var(--text-muted); font-size: var(--fs-small); }
.land-control__result { flex-basis: 100%; font-family: var(--font-mono); font-size: var(--fs-small); color: var(--ink); overflow-wrap: anywhere; white-space: pre-wrap; }
/* the outcome, in the honest palette: a minted PR reads as a confirmed-green link; a
   guard block is a calm dim notice (deliberate friction, not alarm); a failure takes the
   miss hue. */
.land-control__result--ok { color: var(--verified); }
.land-control__result--blocked { color: var(--text-muted); }
.land-control__result--error { color: var(--risk); }
/* post-open lifecycle (DESIGN §29.2: Landed ≠ Merged): a calm "not yet merged" notice, a
   confirmed-green "Merged", or the miss hue for a closed-unmerged PR. */
.land-control__lifecycle { flex-basis: 100%; font-size: var(--fs-small); }
.land-control__lifecycle--landed { color: var(--text-muted); }
.land-control__lifecycle--merged { color: var(--verified); }
.land-control__lifecycle--bounced { color: var(--risk); }
/* a surviving-mutant line in the editor: a calm left-edge accent + a glyph, never
   an alarm — the honest "the tests didn't catch this here" marker. */
.review-survivor-line { background: color-mix(in srgb, var(--signal) 12%, transparent); }
.review-survivor-glyph { background: var(--signal); width: 3px !important; margin-left: 2px; }
.land-row__headline { margin: 0 0 var(--sp-1) 0; font-weight: 600; }
.land-row__detail { margin: 0; color: var(--text-muted); }

/* ---- per-state color: the verdict + integration the Lead reads, legible at a
   glance in the honest-state palette. Color REINFORCES the state the headline
   text already names (strip the CSS and the text still reads it); never an alarm
   red/green, never a gauge. ---- */
/* a real catch / a fully-tested ship-ready line — a thing that happened (calm confirmed) */
.review-card[data-state="catch"] .review-card__headline,
.review-card[data-state="tested"] .review-card__headline { color: var(--verified); }
/* partial progress / oracle still running — pending, not done (working amber) */
.review-card[data-state="partial-catch"] .review-card__headline,
.review-card[data-state="in-flight"] .review-card__headline { color: var(--signal); }
/* the oracle ran and said nothing to catch / no mutable signal — neutral, not a loss */
.review-card[data-state="no-catch"] .review-card__headline,
.review-card[data-state="no-oracle-signal"] .review-card__headline { color: var(--text-muted); }
/* the anchor was lost (rename / edited) — the oracle couldn't follow (muted lost) */
.review-card[data-state="lost-via-rename"] .review-card__headline,
.review-card[data-state="anchor-edited"] .review-card__headline { color: var(--risk); }
/* integration: clean (calm), conflict (muted warn, NOT alarm), checks-red (muted loss), pending (neutral) */
.land-row[data-state="land-clean"] .land-row__headline { color: var(--verified); }
.land-row[data-state="land-conflict"] .land-row__headline { color: var(--signal); }
.land-row[data-state="land-checks-red"] .land-row__headline { color: var(--risk); }
.land-row[data-state="land-pending"] .land-row__headline { color: var(--text-muted); }

/* ---- first-run onboarding affordance: shown only on a truly-fresh session
   (data-state="empty"). A calm guide to the core loop, not an alarm — a quiet
   accent rule, dim supporting text, no animation/gauge (guardrails). ---- */
.onboarding[data-state="empty"] {
  border-left: 2px solid var(--signal);
}
.onboarding__lead { margin: 0 0 var(--sp-1) 0; font-weight: 600; color: var(--ink); }
.onboarding__step { margin: 0 0 var(--sp-1) 0; color: var(--text-muted); font-size: var(--fs-small); }

/* ---- the Spend action: the Lead's core economic move, shown only when there is
   balance to spend. A calm, deliberate control in the balance hue — not an alarm,
   not a pulsing call-to-action. ---- */
.spend-action { margin: 0 0 var(--sp-3) 0; }  /* box CSS on .pk-btn */

/* ---- the prep bench: the fundable work on deck, each target a card the Lead can
   FUND or SHARPEN (split / criteria / convention) during dead-air. A calm stacked
   list, no alarm, no gauge. ---- */
.bench {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-1) var(--sp-5);
  margin-bottom: var(--sp-3);
  border-left: 2px solid var(--hairline);
}
/* .bench__label is a .pk-section-label; each .bench__item is a .pk-card. */
.bench__item { display: flex; flex-direction: column; gap: var(--sp-1); }
.bench__head { display: flex; align-items: baseline; justify-content: space-between; gap: var(--sp-3); }
.bench__target { font-family: var(--font-mono); font-size: var(--fs-small); }
/* the fund affordance spends a catch — balance-hue on hover (the old chip cue). */
.bench__fund:hover { color: var(--signal); border-color: var(--signal); }
/* the sharpen disclosure: a calm dim toggle, not a shouting call-to-action. */
.bench__sharpen {
  cursor: pointer;
  color: var(--text-faint);
  font-size: var(--fs-micro);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.bench__body { display: flex; flex-direction: column; gap: var(--sp-1); padding-top: var(--sp-1); }
.bench__criteria { width: 100%; min-height: 3em; resize: vertical; font-family: var(--font-mono); }
/* an attached sharpening, shown as calm dim lines (a decision the Lead made). */
.bench__anno { display: flex; flex-direction: column; gap: 2px; }
.bench__anno-item { color: var(--text-muted); font-size: var(--fs-small); }

/* the agent-runner control: a calm act-now row (host vs container for live packets),
   sitting with the funding controls it governs. */
.live-runner { display: flex; align-items: baseline; gap: var(--sp-3); }
.live-runner__mode { color: var(--text-muted); font-size: var(--fs-small); }

/* the fleet board's live activity beat — what an agent is doing right now, a calm
   dim ticker on its row (shown only while a packet fills). */
.board-row__activity-beat { color: var(--text-muted); font-size: var(--fs-small); }

/* ---- Flow A: the live card's two sub-landmarks. The split is carried by the
   labelled <section> regions (and the per-row .pk-card elevation), NOT a third
   background layer — a third surface tier is gated out (§1). A little vertical
   rhythm between the regions is the only chrome the calm system needs. ---- */
[role="main"] > section + section { margin-top: var(--sp-5); }

/* ---- Flow B: the unified funding group. Spend (balance hue) and place-packet
   (bandwidth/accent hue) co-located under one label + a dim two-currency
   explainer. A labelled affordance pair — never a meter/gauge/bar. ---- */
.fund-work {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-1) var(--sp-5);
  margin-bottom: var(--sp-3);
  border-left: 2px solid var(--hairline);
}
.fund-work__explainer { color: var(--text-muted); font-size: var(--fs-small); margin: 0; }

/* ---- Flow C: the drill-return affordances on /review and /settings reuse the
   breadcrumb crumb idiom (.board-nav__crumb), so they inherit its calm hue + focus.
   Only the surrounding paragraph needs spacing. ---- */
.review__return, .review__up, .settings__return { margin: 0 0 var(--sp-3) 0; font-size: var(--fs-small); }

/* ---- the Console shell — needs-you rail | preserved center
   column | settled+watches rail, per the Console layout.
   FINAL design-system class names (console, console__*); the pre-existing
   .pk-*/section classes nested inside .console__main are untouched. Every
   region is bounded by a 1px --hairline; cards never drop-shadow. ---- */
.console {
  display: grid;
  grid-template-columns: 360px 1fr 340px;
  align-items: start;
  border: 1px solid var(--hairline);
  border-radius: var(--r-card);
  overflow: hidden;
}
.console__rail {
  display: flex;
  flex-direction: column;
  background: var(--surface-panel);
  min-width: 0;
}
.console__rail--needs-you { border-right: 1px solid var(--hairline); }
.console__rail--settled { border-left: 1px solid var(--hairline); }
.console__panel-header {
  padding: var(--sp-4) var(--sp-6);
  background: var(--surface-card);
  border-bottom: 1px solid var(--hairline);
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  font-weight: 700;
  letter-spacing: var(--track-kicker);
  text-transform: lowercase;
  color: var(--ink);
}
.console__rail-body {
  display: flex;
  flex-direction: column;
  gap: 11px;
  padding: var(--sp-5);
}
.console__card {
  display: block;
  padding: var(--sp-3) var(--sp-4);
  background: var(--surface-card);
  border: 1px solid var(--hairline);
  border-radius: var(--r-card-sm);
  text-decoration: none;
  color: inherit;
}
.console__card--dashed {
  background: transparent;
  border: 1px dashed var(--border-dashed);
  text-align: center;
  color: var(--text-faint);
}
.console__thread-title {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  color: var(--ink);
}
.console__thread-loc {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--text-faint);
  margin-top: var(--sp-1);
}
.console__thread-arrow {
  display: block;
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--signal);
  margin-top: var(--sp-1);
}
.console__more {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--text-faint);
  text-align: center;
}
.console__dry-aside {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--text-ghost);
  text-align: center;
  padding: 0 var(--sp-3);
}
.console__empty-kicker {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  letter-spacing: var(--track-label);
  text-transform: lowercase;
}
.console__main {
  padding: 30px 32px 32px;
  min-width: 0;
}
.console__hero {
  display: flex;
  align-items: baseline;
  gap: var(--sp-6);
  margin-bottom: var(--sp-7);
}
.console__hero-stat {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--fs-hero-stat);
  font-weight: 600;
  color: var(--ink);
  line-height: 1;
}
.console__hero-label { color: var(--text-muted); font-size: var(--fs-body); }
.console__interrupt-kpi {
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--fs-tiny);
  font-weight: 700;
  color: var(--verified);
}
.console__hero-addr {
  margin-left: auto;
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  color: var(--text-faint);
}
.console__cell {
  display: inline-block;
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: var(--r-glyph);
  box-sizing: border-box;
}
.console__cell[data-state="verified"] { background: var(--verified); }
.console__cell[data-state="held"] { background: var(--held); }
/* delivered = ACK'd healthy — the same dark-cyan fill as the mark's BR
   delivered cell, since the story is "outline → delivered fill = promised
   → landed" and this IS that fill, per-packet, on a real ACK. */
.console__cell[data-state="delivered"] { background: var(--delivered); }
/* the blocking hold's matte hue — the same red the needs-you rail pulses,
   but settled rows are retrospective ("live pulses; settled is matte"). */
.console__cell[data-state="held-blocking"] { background: var(--risk); }
/* only WITHIN the needs-you rail does a blocking hold pulse — it is the one
   live, still-actionable use of the color; the settled rail never pulses. */
.console__rail--needs-you .console__cell[data-state="held-blocking"] {
  animation: pk-held-pulse 2s ease-in-out infinite;
}
.console__cell[data-state="in-flight"] { background: var(--signal); animation: pk-pulse 1.8s ease-in-out infinite; }
/* the ghost "composing" cell — the mark's ghost idiom (8%-tint fill,
   delivered-colored stroke) reused for a queued packet row. */
.console__cell[data-state="composing"] {
  background: color-mix(in srgb, var(--delivered) 8%, transparent);
  border: 1.5px solid var(--delivered);
}
.console__settled-row { display: flex; align-items: center; gap: var(--sp-3); }
.console__settled-id { font-family: var(--font-mono); font-size: var(--fs-tiny); color: var(--ink); }
.console__settled-outcome { margin-left: auto; font-family: var(--font-mono); font-size: var(--fs-tiny); color: var(--text-muted); }
.console__inflight {
  margin-top: var(--sp-6);
  padding-top: var(--sp-5);
  border-top: 1px solid var(--hairline);
}
.console__inflight-row {
  display: flex;
  align-items: center;
  gap: var(--sp-4);
  padding: var(--sp-2) 0;
}
.console__inflight-name { font-family: var(--font-mono); font-size: var(--fs-tiny); color: var(--ink); }
.console__inflight-intent { font-family: var(--font-mono); font-size: var(--fs-micro); color: var(--text-muted); }

/* ---- lane health — 4 kicker+count cards, tallied ONLY
   from the session's lane cache. Neutral hairline cards, no state colors
   (lane is QoS, not a lifecycle state), no shadows. ---- */
.console__lane-health {
  margin-top: var(--sp-6);
  padding-top: var(--sp-5);
  border-top: 1px solid var(--hairline);
}
.console__lane-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-3);
  margin-top: var(--sp-3);
}
.console__lane-card {
  padding: var(--sp-3) var(--sp-4);
  background: var(--surface-card);
  border: 1px solid var(--hairline);
  border-radius: var(--r-card-sm);
  text-align: center;
}
.console__lane-kicker {
  display: block;
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  letter-spacing: var(--track-label);
  text-transform: lowercase;
  color: var(--text-faint);
}
.console__lane-count {
  display: block;
  margin-top: var(--sp-1);
  font-family: var(--font-mono);
  font-variant-numeric: tabular-nums;
  font-size: var(--fs-emph);
  font-weight: 600;
  color: var(--ink);
}

/* "your watches" — three canonical standing triggers, each
   a neutral card (lanes are QoS, not states; watches are the same — no
   state-grammar color belongs here, only text tokens). */
.console__watch-name {
  font-family: var(--font-mono);
  font-size: var(--fs-tiny);
  color: var(--ink);
}
.console__watch-precision {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--text-faint);
  margin-top: var(--sp-1);
}
.console__watch-prompt {
  display: flex;
  align-items: baseline;
  gap: var(--sp-2);
  margin-top: var(--sp-2);
}
.console__watch-prompt-name {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  color: var(--text-muted);
  flex: 1;
}
.console__watch-mark {
  font-size: var(--fs-micro);
  padding: 1px var(--sp-3);
}

/* ---- the Inspector shell — identity strip | changed-files
   tree | Monaco island + answer form | annotation rail | timeline footer, per
   the Inspector layout. FINAL design-system class names
   (inspector, inspector__*, annotation-card*); .review-thread/.review-answer/
   .review-adjust/.file-tree keep their own rules above (additive multi-class).
   Every region is hairline-bounded; cards never drop-shadow. ---- */
.inspector__titlebar {
  display: flex;
  align-items: baseline;
  gap: var(--sp-5);
  padding-bottom: var(--sp-3);
  margin-bottom: var(--sp-5);
  border-bottom: 1px solid var(--hairline);
  font-family: var(--font-mono);
}
.inspector__name { color: var(--ink); font-weight: 600; font-size: var(--fs-body-mono); }
.inspector__packet-name { color: var(--text-muted); font-size: var(--fs-tiny); }
.inspector__rev { color: var(--text-muted); font-size: var(--fs-tiny); }
/* a NEUTRAL pill — lane is a QoS class, never a lifecycle
   state, so it never borrows the state-grammar colors (--held/--risk/etc).
   Same flat-raised idiom as the annotation card's neutral chips. */
.inspector__lane {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  letter-spacing: var(--track-chip);
  text-transform: lowercase;
  padding: 1px var(--sp-3);
  border-radius: var(--r-pill);
  border: 1px solid var(--hairline);
  background: var(--surface-raised);
  color: var(--text-faint);
}
.inspector__addr { margin-left: auto; color: var(--text-faint); font-size: var(--fs-tiny); }
.inspector {
  display: grid;
  grid-template-columns: 252px 1fr 312px;
  align-items: start;
  border: 1px solid var(--hairline);
  border-radius: var(--r-card);
  overflow: hidden;
}
.inspector__tree {
  padding: var(--sp-5);
  background: var(--surface-panel);
  border-right: 1px solid var(--hairline);
  min-width: 0;
}
.inspector__main {
  padding: var(--sp-5);
  background: var(--surface-card);
  min-width: 0;
}
.inspector__rail {
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  padding: var(--sp-5);
  background: var(--surface-panel);
  border-left: 1px solid var(--hairline);
  min-width: 0;
}
.inspector__rail-header {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  font-weight: 700;
  letter-spacing: var(--track-kicker);
  text-transform: lowercase;
  color: var(--text-muted);
  padding-bottom: var(--sp-2);
  border-bottom: 1px solid var(--hairline);
}
.inspector__tree-empty {
  padding: var(--sp-4);
  border: 1px dashed var(--border-dashed);
  border-radius: var(--r-card-sm);
  text-align: center;
  font-family: var(--font-mono);
  font-size: var(--fs-small);
  color: var(--text-faint);
}
.inspector__timeline {
  margin-top: var(--sp-5);
  padding: var(--sp-4);
  border: 1px solid var(--hairline);
  border-radius: var(--r-card-sm);
}
.inspector__timeline-kicker {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  font-weight: 700;
  letter-spacing: var(--track-kicker);
  text-transform: lowercase;
  color: var(--text-faint);
  margin-bottom: var(--sp-1);
}
/* the gauntlet's six-gate record — one row per gate, a
   neutral name label, a status pill (color-mix idiom, same shape as the
   annotation card's chips), and an honest detail note. Status colors are
   the SAME state-grammar colors used everywhere else (--verified/--held/
   --risk/--text-faint) — never invented. */
.gauntlet__list {
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}
.gauntlet-gate {
  display: flex;
  align-items: baseline;
  gap: var(--sp-3);
  font-family: var(--font-mono);
  font-size: var(--fs-small);
}
.gauntlet-gate__name { color: var(--text-muted); min-width: 11em; }
.gauntlet-gate__pill {
  font-size: var(--fs-micro);
  letter-spacing: var(--track-chip);
  text-transform: lowercase;
  padding: 1px var(--sp-3);
  border-radius: var(--r-pill);
  border: 1px solid var(--hairline);
  color: var(--text-faint);
}
.gauntlet-gate__pill[data-status="passed"] {
  color: var(--verified);
  border-color: color-mix(in srgb, var(--verified) 40%, var(--hairline));
  background: color-mix(in srgb, var(--verified) 14%, transparent);
}
.gauntlet-gate__pill[data-status="failed"] {
  color: var(--risk);
  border-color: color-mix(in srgb, var(--risk) 40%, var(--hairline));
  background: color-mix(in srgb, var(--risk) 14%, transparent);
}
.gauntlet-gate__pill[data-status="held"] {
  color: var(--held);
  border-color: color-mix(in srgb, var(--held) 40%, var(--hairline));
  background: color-mix(in srgb, var(--held) 14%, transparent);
}
.gauntlet-gate__detail { color: var(--text-faint); font-size: var(--fs-tiny); }
.gauntlet-gate__confirm { margin-left: var(--sp-2); }
/* the annotation card (the design system's AnnotationCard spec): authorship is the 3px
   left border — --agent for the oracle findings that dominate the rail.
   Durable human annotations reuse the same card with their own author chip.
   Overrides .review-thread's own thinner signal-hue accent (later in the
   stylesheet wins at equal specificity). */
.annotation-card {
  border-left: 3px solid var(--agent);
  background: var(--surface-raised);
  border-radius: var(--r-ann);
  padding: var(--sp-3) var(--sp-4);
}
.annotation-card__head { display: flex; align-items: center; gap: var(--sp-2); margin-bottom: var(--sp-2); }
.annotation-card__chip {
  font-family: var(--font-mono);
  font-size: var(--fs-micro);
  letter-spacing: var(--track-chip);
  text-transform: lowercase;
  padding: 1px var(--sp-3);
  border-radius: var(--r-pill);
  border: 1px solid var(--hairline);
  color: var(--text-muted);
}
.annotation-card__chip--author {
  color: var(--agent);
  border-color: color-mix(in srgb, var(--agent) 40%, var(--hairline));
  background: color-mix(in srgb, var(--agent) 14%, transparent);
}
.annotation-card__chip--sev {
  color: var(--held);
  border-color: color-mix(in srgb, var(--held) 40%, var(--hairline));
  background: color-mix(in srgb, var(--held) 14%, transparent);
}
.annotation-card__where { margin-left: auto; color: var(--text-faint); font-size: var(--fs-tiny); }
.annotation-card__reply { margin-top: var(--sp-2); padding-left: var(--sp-2); border-left: 1px solid var(--hairline); display: flex; flex-direction: column; gap: var(--sp-1); }
.annotation-card__reply .annotation-card__chip--author { align-self: flex-start; }
.annotation-card__reply-form { margin-top: var(--sp-2); display: flex; gap: var(--sp-1); }
.annotation-card__reply-input { flex: 1; font-size: var(--fs-small); }
.annotation-card__reply-btn { flex: none; }
`
