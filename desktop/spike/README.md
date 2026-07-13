# spike — desktop walkthrough

**Shock the system.** An open-model arrival rate sits at a baseline, then jumps ~10× for a sustained surge before dropping back. It answers two questions: does it survive the spike, and does it recover cleanly afterwards?

| Screenshot | What's happening |
|---|---|
| <img src="00-canvas.png" width="460"> | **Open the plan into the Canvas.** The plan renders as a drag-and-drop node graph — `PLAN ▸ SCENARIO ▸` its steps — with the step palette on the left. |
| <img src="01-inspect-request.png" width="460"> | **Click a request node.** Its full GUI form opens in the inspector — method, URL, headers, body, and the checks/extracts. No YAML required. |
| <img src="02-inspect-scenario.png" width="460"> | **Click the scenario node.** The workload shape is a form too — the executor and its parameters, right where you dial the load. `ramping-arrival-rate` — baseline → spike → sustained surge → drop → verify recovery, with a pre-allocated VU pool that can burst. |
| <img src="03-yaml.png" width="460"> | **Flip to YAML.** Always in sync with the canvas and byte-for-byte what `loadr run` executes in CI — `base_url` stays an `${env.BASE_URL}` reference. The desktop app is a lens over the same engine, not a re-implementation. |
| <img src="04-run-live.png" width="460"> | **Hit Run.** loadr Desktop drives the plan against the live storefront with the bundled engine while the graph stays in view; metrics stream in real time. The request-rate chart steps up hard at the spike, then settles as it recovers. |
| <img src="05-results.png" width="460"> | **Read the verdict.** Headline tiles settle and the result pills report every gate — thresholds, checks, error rate and latency percentiles. **Export JUnit** is the CI report. Judged under the surge (**p95 < 2.5 s** on a small CI runner taking a 10× order spike). |
| <img src="06-history.png" width="460"> | **Run it again.** Each run drops into **History · compare** — pick any earlier run as a baseline and loadr Desktop diffs the two, so a regression shows up the moment it appears. |

---

<sub>Captured against the demo storefront (`make db` + `make api`) with the desktop capture rig (see [`../tools`](../tools)). Long durations are clamped so the live run finishes on-screen; the scenario shape, checks and thresholds are the plan's own. Source plan: [`../../perf/spike.yaml`](../../perf/spike.yaml).</sub>
