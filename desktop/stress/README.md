# stress — desktop walkthrough

**Push past the knee.** VUs ramp from zero, up past the comfortable point, then back down. The goal isn't to pass — it's to find the load at which latency runs away, and to confirm the service recovers on the way down.

| Screenshot | What's happening |
|---|---|
| <img src="00-canvas.png" width="460"> | **Open the plan into the Canvas.** The plan renders as a drag-and-drop node graph — `PLAN ▸ SCENARIO ▸` its steps — with the step palette on the left. The scenario is a **ramping-vus** stage list; the requests it drives hang off it. |
| <img src="01-inspect-request.png" width="460"> | **Click a request node.** Its full GUI form opens in the inspector — method, URL, headers, body, and the checks/extracts. No YAML required. |
| <img src="02-inspect-scenario.png" width="460"> | **Click the scenario node.** The workload shape is a form too — the executor and its parameters, right where you dial the load. `ramping-vus`; its stage list (warm up → push past the knee → ramp down) sits under **Raw (YAML)** and reads in full in the YAML view. |
| <img src="03-yaml.png" width="460"> | **Flip to YAML.** Always in sync with the canvas and byte-for-byte what `loadr run` executes in CI — `base_url` stays an `${env.BASE_URL}` reference. The desktop app is a lens over the same engine, not a re-implementation. |
| <img src="04-run-live.png" width="460"> | **Hit Run.** loadr Desktop drives the plan against the live storefront with the bundled engine while the graph stays in view; metrics stream in real time. Watch active VUs climb the stages and p95 stretch as the service passes its knee. |
| <img src="05-results.png" width="460"> | **Read the verdict.** Headline tiles settle and the result pills report every gate — thresholds, checks, error rate and latency percentiles. **Export JUnit** is the CI report. A deliberately loose **p95 < 3 s** — this stage is about the shape of the curve, not a tight SLO. |
| <img src="06-history.png" width="460"> | **Run it again.** Each run drops into **History · compare** — pick any earlier run as a baseline and loadr Desktop diffs the two, so a regression shows up the moment it appears. |

---

<sub>Captured against the demo storefront (`make db` + `make api`) with the desktop capture rig (see [`../tools`](../tools)). Long durations are clamped so the live run finishes on-screen; the scenario shape, checks and thresholds are the plan's own. Source plan: [`../../perf/stress.yaml`](../../perf/stress.yaml).</sub>
