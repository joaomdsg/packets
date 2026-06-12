# Round 99 — order-loop slice B: verify the Monaco loop in a real browser — 2026-06-12

Trigger: every Monaco island in this codebase (R62 review, R96/R98 authoring) is
browser-verified by hand, never automated — unit tests stop at the server contract.
The R98 close-out named an end-to-end browser run of author→analyze→place as the
one unautomatable check. This round automates it against a real headless Chrome.

## The change

`internal/app/authoring_browser_test.go` (build tag `browser`, so it never runs in
normal CI) boots the live server, launches headless Chrome for Testing, and drives
the authoring loop:

- It serves Monaco from disk by intercepting `cdn.jsdelivr.net` (unreachable behind
  the egress allowlist) through the CDP `fetch` domain and fulfilling each
  `min/vs/*` request from `MONACO_VS` — so the editor actually mounts.
- It waits for the real `.monaco-editor`, types a draft, waits out the 900ms
  debounce, and asserts the producer's (stubbed) summary rendered back through the
  bridge → action → SSE re-render path, AND that the flagged span decorated the
  editor inline (`.authoring-flag-question` count > 0).

The producer is stubbed (`analyzeDraft`) because the subject is the editor + bridge
+ render round-trip, not a live claude run. The test self-skips unless `CFT_CHROME`
and `MONACO_VS` point at a Chrome binary and the Monaco assets.

This added `chromedp` (+ cdproto) to the module. The browser tag keeps it out of
every normal build and test run; it is a test-only, opt-in dependency.

## The result

With Chrome for Testing 147 (downloaded from `storage.googleapis.com`) and Monaco
0.52.2 (from the npm registry), the test passes in ~4s: the editor mounts, typing
drives the debounced analysis through the datastar bridge, the server runs the
producer, and the read renders back with the flagged span underlined inline. A
screenshot is written to `/tmp/authoring-browser.png` for the human record.

## New clashes opened / resolved

Resolved: the Monaco authoring loop is now verified end-to-end in a real browser,
not only by the server-side unit tests — closing the R96/R98 "browser-verified by
hand" caveat for the authoring surface. Open: Monaco is still loaded from a CDN at
runtime (the test only proves it works when the assets are reachable); vendoring
the assets behind a server `/static` handler — already flagged in the review island
as a future hardening slice — would make the editor work behind a strict egress
allowlist in production, not just in the intercepting test.
