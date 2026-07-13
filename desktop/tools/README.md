# desktop walkthrough tooling

The walkthroughs in `desktop/<demo>/` are **captured, not mocked**: a scripted
[loadr Desktop](https://loadr.io/download/#desktop) session opens each plan from
`perf/`, walks it through the canvas, and runs it live against the demo
storefront, screenshotting every transition.

| Piece | What it does |
|---|---|
| `capture.sh` | One command to re-capture any (or all) demos: preps a capture copy of the plan, drives the app, drops the screenshots into `desktop/<demo>/`, regenerates the READMEs. |
| `build-readme.py` | Renders each demo's `README.md` — pairs the captured `NN-slug.png` screenshots with the per-demo narrative kept in this file. |
| the driver | Lives in the main loadr repo (`desktop/e2e/demo-walkthroughs.mjs` + `desktop/e2e/prep-demo-plan.py`) next to the app it drives — Playwright-for-Electron, launched with the plan path, output dir and demo name. |

## The capture copy

The screenshots must show the *real* plan, but a 5-minute soak can't finish
on-screen. `prep-demo-plan.py` makes exactly three mechanical edits to a copy:

1. **durations are clamped** (default 8s per stage) so live runs finish while
   the ramp shapes stay intact;
2. **relative `data/` feeder paths are absolutized** (the app runs plans from a
   temp dir);
3. **a hardcoded `base_url` is retargeted** at the live backend when asked
   (`--base-url`); plans using `${env.BASE_URL}` are left alone — the env var
   does the job, exactly as in CI.

Scenario shapes, checks and thresholds are never touched — the verdicts in the
screenshots are the plans' own gates, evaluated for real.

## Re-capturing

```bash
make db && make api                      # backend on :8080
make observe-up                          # only needed for observe-mixed
BASE_URL=http://localhost:8080 desktop/tools/capture.sh          # all nine
BASE_URL=http://localhost:8080 desktop/tools/capture.sh journey  # just one
```

Headless (CI): prefix with `xvfb-run -a`.
