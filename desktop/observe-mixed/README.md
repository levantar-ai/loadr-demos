# observe-mixed — desktop walkthrough

**Load ↔ system, side by side.** loadr's `observe:` block pulls host and datastore metrics (CPU, memory, load average, Postgres connections/commits, Redis ops) from Prometheus while the load runs — so a latency bump lines up against the resource that caused it, instead of you eyeballing two dashboards.

| Screenshot | What's happening |
|---|---|
| <img src="00-canvas.png" width="460"> | **Open the plan into the Canvas.** The plan renders as a drag-and-drop node graph — `PLAN ▸ SCENARIO ▸` its steps — with the step palette on the left. A `ramping-vus` scenario over a mixed read/write flow, with the `observe:` sources attached at the plan level. |
| <img src="01-inspect-request.png" width="460"> | **Click a request node.** Its full GUI form opens in the inspector — method, URL, headers, body, and the checks/extracts. No YAML required. |
| <img src="02-inspect-scenario.png" width="460"> | **Click the scenario node.** The workload shape is a form too — the executor and its parameters, right where you dial the load. `ramping-vus` — ramp up, then hold, so the correlation has a steady window. |
| <img src="03-yaml.png" width="460"> | **Flip to YAML.** Always in sync with the canvas and byte-for-byte what `loadr run` executes in CI — `base_url` stays an `${env.BASE_URL}` reference. The desktop app is a lens over the same engine, not a re-implementation. |
| <img src="04-run-live.png" width="460"> | **Hit Run.** loadr Desktop drives the plan against the live storefront with the bundled engine while the graph stays in view; metrics stream in real time. As VUs ramp, the system metrics move with them — that alignment is the whole point. |
| <img src="05-results.png" width="460"> | **Read the verdict.** Headline tiles settle and the result pills report every gate — thresholds, checks, error rate and latency percentiles. **Export JUnit** is the CI report. The summary carries the correlated system series alongside the load metrics; `loadr report` renders the overlay. |
| <img src="06-history.png" width="460"> | **Run it again.** Each run drops into **History · compare** — pick any earlier run as a baseline and loadr Desktop diffs the two, so a regression shows up the moment it appears. |

---

<sub>Captured against the demo storefront (`make db` + `make api`) with the desktop capture rig (see [`../tools`](../tools)). Long durations are clamped so the live run finishes on-screen; the scenario shape, checks and thresholds are the plan's own. Source plan: [`../../perf/observe-mixed.yaml`](../../perf/observe-mixed.yaml).</sub>
