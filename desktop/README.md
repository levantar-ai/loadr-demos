# Desktop walkthroughs

Every perf demo in [`perf/`](../perf) — opened, inspected and **run live** in
[loadr Desktop](https://loadr.io/download/#desktop), one screenshot per
transition with narrative and direction. Each walkthrough leads with the
**canvas**: the plan as a drag-and-drop node graph, every node editable through
a GUI form, and the same bundled engine CI uses running the plan underneath.

| Walkthrough | The idea |
|---|---|
| [`smoke/`](smoke/) | The fast gate — 2 VUs, strict checks; if this fails, stop. |
| [`load/`](load/) | Sustained nominal traffic — the everyday baseline. |
| [`stress/`](stress/) | Ramp past the knee — find where it breaks, watch it recover. |
| [`spike/`](spike/) | A sudden ~10× surge, then recovery. |
| [`soak/`](soak/) | Hold steady for a long time — catch leaks and drift. |
| [`impulse/`](impulse/) | Every VU at once — thundering herd, cold caches. |
| [`arrival-rate/`](arrival-rate/) | Open model — fix the throughput, not the VUs. |
| [`journey/`](journey/) | Login → browse → create → order → verify, with CSV feeders, extraction and correlation. |
| [`plugin-postgres/`](plugin-postgres/) | Skip the app — load-test PostgreSQL directly; the SQL query is the request. |
| [`observe-mixed/`](observe-mixed/) | Overlay Prometheus system metrics on the load — correlation, not guesswork. |

Each directory is a README with a two-column table — **screenshot on the left,
what's happening on the right** — captured for real against the demo storefront
by [`tools/`](tools/).
