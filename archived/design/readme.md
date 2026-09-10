# Packets — Design System

Packets is a control plane for agent-written code changes. Agents compose, verify, and forward changes ("packets") at line rate against machine-checkable contracts ("handshakes"); the few with real blast radius are **held** for a human. The product borrows its entire mental model from networking: addrs (repos, `owner/repo` form), lanes (best-effort → standard → strict → irreversible), forwarding, inspection, ACK-on-delivery.

**Source of truth:** this design system is itself the canonical spec — tokens, brand rules, voice, and a component catalog encode the settled designs verbatim. Exact values (colors, px, radii, tracking) live in `tokens/*.css`; port those first and never invent a new state color.

## The state grammar (the most important thing here)

Packet state is a color system, and it is used *everywhere* — chips, dots, cells, the logo itself:

| state | visual | token |
|---|---|---|
| composing | **outline** (stroke, faint fill) | `--delivered` stroke @ 18% of cell |
| in flight / you | bright cyan fill | `--signal` #4cc4d4 |
| verified / forwarded | green | `--verified` #46c08a |
| held (advisory, sampled) | amber | `--held` #e6b23e |
| held (blocking: strict, guardrail, irreversible) | red | `--risk` #f0666b |
| agent authorship | purple | `--agent` #a78bfa |
| delivered | dark cyan fill | `--delivered` #2a7683 |

Rules: outline → fill = promised → landed (same color at both ends = same addr). Live things pulse or glow; settled things are matte. Never invent a new state color.

## The mark (locked spec)

2×2 grid of packet cells. TL/BL = signal fill; TR = ghost (composing): stroke 18% of cell width in `--delivered`, 8% tint fill, scale 1.035; BR = `--delivered` solid fill. Gap = 27% of cell; radius = 23% of cell. **Small-size rule:** below ~14px cell, the ghost outline is forbidden — use solid `--delivered-mid` #357f8a. Silhouette over story. Wordmark: IBM Plex Mono 700 lowercase, +.04em, lockup gap ≈ 1.2× cell. In-app chrome uses the mark alone or the stacked `packets / INSPECTOR` lockup — never mark + wordmark + breadcrumb together.

## CONTENT FUNDAMENTALS

- **Mono is the machine's voice.** Everything operational — labels, chips, addrs, counts, timestamps, meta — is Plex Mono. Sans appears only in prose sentences (annotation bodies, marketing paragraphs).
- **Lowercase by default.** Kickers and panel labels are lowercase or UPPERCASE-tracked mono; sentences start lowercase in meta ("held 34m", "the whole job"). Marketing headlines use sentence case.
- **Networking vocabulary, always:** forward, hold, inspect, deliver, ACK, addr, lane, packet, stream, line rate. Never "PR", "merge queue", "approve", "review" (as a noun).
- **No exclamation marks. No emoji.** Unicode glyphs are the icon set (see ICONOGRAPHY).
- **Second person, direct:** "needs you · 4", "nothing needs you", "the one packet the loop held today". The system reports to *you*.
- **Counts are stated plainly** with the `·` separator: "4 files · +85 −6", "conf 0.72", "seq 01–03 / 05".
- **The tone is calm operator confidence** — declarative, unhurried, a little dry: "An empty queue is success, not idleness." "green means gone."

## VISUAL FOUNDATIONS

- **Color:** near-black blue ground (#0a0d13) everywhere; four surface steps up (#080a0f deep / #0f131b panel / #141924 card / #1b212e raised). One accent family (cyan) plus the semantic state set. Marketing adds cream #F6EBCD display headings — never in-app.
- **Type:** IBM Plex Sans + Plex Mono (Google Fonts). Dense mono scale 9–13px in-app; tabular numerals always. Display sizes only on marketing surfaces (36–64px, negative tracking).
- **Spacing:** dense. 7–14px paddings in cards; 12×16 panel headers; 22–32px page gutters. Nothing floats — every region is bounded by a 1px hairline (#1b212e or rgba(255,255,255,.08) on panels).
- **Backgrounds:** flat fills only, plus (sparingly) a cyan radial wash at a top corner of main content (`--wash-signal`). No gradients otherwise, no imagery, no textures.
- **Cards:** `--surface-card` fill, 1px faint border, 8–12px radius, NO drop shadow. Depth is reserved for full app-frame mocks (`--shadow-frame`) and tooltips. Annotation cards carry a 3px left border in the author's color — this left-accent pattern is native to this brand (authorship), don't generalize it to other card types.
- **Chips:** pill-shaped mono microtype (9–10px, 700, uppercase for states), tinted via `color-mix(state 13–16%, surface)`, optional 1px border at 40–45% mix. Neutral chips are flat `--surface-raised`.
- **Glows & motion:** live = pulsing dot (pk-pulse, 1.8–2s) or soft glow; HELD = red glow pulse (pk-held-pulse); stream activity = pk-flow shimmer on a 3px track. Transitions ~.2s on interactive, .4s on state recolor. No bounces, no slides.
- **Hover states:** border brightens toward the semantic color + text tints (chips), or border-color shift on buttons. No scale transforms.
- **Empty states:** dashed 1px border (#3a4150 or faint), flat ground fill, centered mono microtype. Empty is a *good* state here — copy celebrates it.
- **Progress:** 5–6px tracks, `--surface-raised` ground, semantic fill, 3px radius.
- **Data-viz:** bar charts are 1px-gapped vertical bars in `--verified`; held markers are 4px amber ticks. Axis labels 9px mono ghost.
- **Corner radii:** see tokens/spacing.css; chips are pills; app shells 16px.
- **Transparency/blur:** none. Opaque surfaces only.

## ICONOGRAPHY

No icon font, no SVG icon set. The brand's icons are **unicode glyphs set in Plex Mono**: ✎ (you/annotate), ⚑ (flag/guardrail), ● (live state), ◂ ▸ (pagers), ⧉ (scoped), ✱ (aside), ⇄ (split), ＋ (new), ✕ (clear), → (links, always trailing), ⌁ (held), ✓ (verified). Dots and 8px squares (border-radius 2px) are the packet-state markers. **The 15px rounded square with a letter (M/A/⚑) is the file-status glyph.** Never import an icon library; never draw SVG icons. The one piece of vector art in the project is the mark itself (`assets/branding/mark.svg`).

No raster logo asset exists — the mark ships as a canonical SVG (vector, never rasterized), and in-app it's constructed in code (see PacketMark component) so it can carry live state. Both express the same locked recipe.

## Index

- `styles.css` → `tokens/` (colors, typography, spacing, effects, base, fonts) — the visual-identity contract; port these values verbatim
- `guidelines/` — the concept model (`concepts.md`), intellectual lineage (`lineage.md`), and copy voice (`voice.md`)
- `components/` — a per-component catalog (`*.prompt.md`): purpose, variants, and usage for PacketCell, PacketMark, Chip, Button, Card, AnnotationCard, Titlebar, Timeline, Terminal
- `assets/branding/mark.svg` — the canonical packets mark (vector)
