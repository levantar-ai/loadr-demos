#!/usr/bin/env python3
"""Render a desktop-walkthrough README for each perf demo.

Pairs the captured screenshots (NN-slug.png, produced by the capture rig) with
per-demo narrative, in a two-column table: screenshot left, description right.

    python3 build-readme.py [demo ...]      # default: every demo with a narrative

The screenshots are the source of truth for *what* the app did; this file is the
source of truth for the *words*. Numbers are left to the screenshots so a
re-capture never dates the prose.
"""
import re
import sys
from pathlib import Path

DESKTOP = Path(__file__).resolve().parent.parent          # loadr-demos/desktop
IMG_W = 460

# Generic direction per step — the action a presenter takes. The demo-specific
# "why it matters" is appended from NARRATIVES[demo]["notes"][slug].
ACTION = {
    "canvas": "**Open the plan into the Canvas.** The plan renders as a drag-and-drop node "
              "graph — `PLAN ▸ SCENARIO ▸` its steps — with the step palette on the left.",
    "inspect-request": "**Click a request node.** Its full GUI form opens in the inspector — "
                       "method, URL, headers, body, and the checks/extracts. No YAML required.",
    "inspect-scenario": "**Click the scenario node.** The workload shape is a form too — the "
                        "executor and its parameters, right where you dial the load.",
    "yaml": "**Flip to YAML.** Always in sync with the canvas and byte-for-byte what `loadr run` "
            "executes in CI — `base_url` stays an `${env.BASE_URL}` reference. The desktop app is a "
            "lens over the same engine, not a re-implementation.",
    "run-live": "**Hit Run.** loadr Desktop drives the plan against the live storefront with the "
                "bundled engine while the graph stays in view; metrics stream in real time.",
    "results": "**Read the verdict.** Headline tiles settle and the result pills report every gate — "
               "thresholds, checks, error rate and latency percentiles. **Export JUnit** is the CI report.",
    "history": "**Run it again.** Each run drops into **History · compare** — pick any earlier run as a "
               "baseline and loadr Desktop diffs the two, so a regression shows up the moment it appears.",
}

FOOTER = (
    "<sub>Captured against the demo storefront (`make db` + `make api`) with the desktop capture rig "
    "(see [`../tools`](../tools)). Long durations are clamped so the live run finishes on-screen; the "
    "scenario shape, checks and thresholds are the plan's own. Source plan: "
    "[`../../perf/{src}`](../../perf/{src}).</sub>"
)

# Per-demo narrative. `notes` are appended to the generic ACTION for that slug.
NARRATIVES = {
    "smoke": {
        "title": "smoke — desktop walkthrough",
        "tagline": "The fast gate: a couple of VUs, a few seconds, strict checks.",
        "intro": "The **fast gate**. A couple of VUs for a few seconds across the storefront's core "
                 "read paths. If `smoke` fails the build is broken and the heavier stages don't run — "
                 "so it's the first thing you'd open, inspect and run in loadr Desktop.",
        "src": "smoke.yaml",
        "notes": {
            "canvas": "Here: `sanity` on `constant-vus` feeding four read requests (`/healthz`, `/readyz`, "
                      "`/api/products`, `/api/products/1`).",
            "inspect-scenario": "`constant-vus`, 2 VUs — barely any load, by design.",
            "run-live": "The gate barely breaks a sweat: thousands of req/s at a ~1 ms p95.",
            "results": "**✓ passed** — zero failures, tens of thousands of checks green, sub-ms p99.",
        },
    },
    "load": {
        "title": "load — desktop walkthrough",
        "tagline": "Sustained nominal traffic — the everyday baseline.",
        "intro": "**Expected traffic, held steady.** A pool of VUs hitting a realistic mix for a sustained "
                 "window — the baseline you compare every other run against. Tight duration thresholds "
                 "(p95 and p99) keep it honest.",
        "src": "load.yaml",
        "notes": {
            "inspect-scenario": "`constant-vus` — a fixed pool held flat for the whole window.",
            "run-live": "A steady state: request rate and active VUs plateau rather than climb.",
            "results": "The run is judged on **p95 < 400 ms and p99 < 800 ms** — the everyday SLO.",
        },
    },
    "stress": {
        "title": "stress — desktop walkthrough",
        "tagline": "Ramp past the knee — find where it breaks.",
        "intro": "**Push past the knee.** VUs ramp from zero, up past the comfortable point, then back down. "
                 "The goal isn't to pass — it's to find the load at which latency runs away, and to confirm "
                 "the service recovers on the way down.",
        "src": "stress.yaml",
        "notes": {
            "canvas": "The scenario is a **ramping-vus** stage list; the requests it drives hang off it.",
            "inspect-scenario": "`ramping-vus`; its stage list (warm up → push past the knee → ramp down) "
                                "sits under **Raw (YAML)** and reads in full in the YAML view.",
            "run-live": "Watch active VUs climb the stages and p95 stretch as the service passes its knee.",
            "results": "A deliberately loose **p95 < 3 s** — this stage is about the shape of the curve, "
                       "not a tight SLO.",
        },
    },
    "spike": {
        "title": "spike — desktop walkthrough",
        "tagline": "A sudden 10× surge, then recovery.",
        "intro": "**Shock the system.** An open-model arrival rate sits at a baseline, then jumps ~10× for a "
                 "sustained surge before dropping back. It answers two questions: does it survive the spike, "
                 "and does it recover cleanly afterwards?",
        "src": "spike.yaml",
        "notes": {
            "inspect-scenario": "`ramping-arrival-rate` — baseline → spike → sustained surge → drop → verify "
                                "recovery, with a pre-allocated VU pool that can burst.",
            "run-live": "The request-rate chart steps up hard at the spike, then settles as it recovers.",
            "results": "Judged under the surge (**p95 < 2.5 s** on a small CI runner taking a 10× order spike).",
        },
    },
    "soak": {
        "title": "soak — desktop walkthrough",
        "tagline": "Hold steady for a long time — catch leaks and drift.",
        "intro": "**The long, boring one — on purpose.** A modest, constant load held for minutes. Nothing "
                 "should change: if latency creeps up or memory drifts, you've found a leak that a short run "
                 "would never surface.",
        "src": "soak.yaml",
        "notes": {
            "inspect-scenario": "`constant-vus` — a small pool held flat, but for a *long* window.",
            "run-live": "The point is what *doesn't* move — the metrics should stay flat the whole way.",
            "results": "Judged on staying flat: **avg < 200 ms and p95 < 500 ms** end to end.",
        },
    },
    "impulse": {
        "title": "impulse — desktop walkthrough",
        "tagline": "Every VU at once — the thundering herd.",
        "intro": "**No ramp.** Every VU starts in the same instant and hammers the service cold — a "
                 "thundering-herd / cold-cache test. It stresses connection pools and cache warm-up in a way "
                 "a gentle ramp hides.",
        "src": "impulse.yaml",
        "notes": {
            "inspect-scenario": "`constant-vus` with the whole pool live from t=0 — no warm-up.",
            "run-live": "Active VUs jump to full immediately; the first seconds are the interesting ones.",
            "results": "Shows how the service handles a cold, simultaneous start rather than a gradual one.",
        },
    },
    "arrival-rate": {
        "title": "arrival-rate — desktop walkthrough",
        "tagline": "Open model — fix the throughput, not the VUs.",
        "intro": "**Requests per second, not users.** An open-model executor holds a fixed iteration rate and "
                 "allocates VUs as needed to sustain it — so throughput stays constant even when responses "
                 "slow, exactly like real inbound traffic.",
        "src": "arrival-rate.yaml",
        "notes": {
            "inspect-scenario": "`constant-arrival-rate` — a target rate with a pre-allocated VU pool that "
                                "grows toward a cap to hold the rate.",
            "run-live": "Request rate pins to the target; active VUs float up and down to keep it there.",
            "results": "Judged on tail latency at a fixed rate — **p99 < 500 ms**.",
        },
    },
    "journey": {
        "title": "journey — desktop walkthrough",
        "tagline": "A real authenticated flow: login → browse → create → order → verify.",
        "intro": "**A realistic user, end to end.** Three loadr features at once — **data feeders** (each login "
                 "pulls credentials from a CSV), **extraction** (pull the bearer token, a SKU and an order id "
                 "from responses), and **correlation** (feed those back into later requests). This is where "
                 "the inspector earns its keep.",
        "src": "journey.yaml",
        "notes": {
            "canvas": "The flow is a chain of requests — login, browse, create, order, verify — each feeding "
                      "the next.",
            "inspect-request": "Open the `login` request to see the CSV-fed body and the `token` **extract**; "
                               "later requests reference `${token}` — correlation, visualised.",
            "inspect-scenario": "`constant-vus` shoppers, each running the full chain with think time between "
                                "steps.",
            "run-live": "Every VU walks the whole journey; failures here mean a broken step in the chain.",
            "results": "Judged on the full authenticated path (**p95 < 600 ms**), proving feeders, extraction "
                       "and correlation all held under load.",
        },
    },
    "plugin-postgres": {
        "title": "plugin-postgres — desktop walkthrough",
        "tagline": "Skip the app — load-test PostgreSQL directly via the postgres plugin.",
        "intro": "**No HTTP at all.** Every other demo drives the Go API; this one hammers the same "
                 "Postgres the app uses, directly — the parameterised **SQL query is the request**, "
                 "pooled per VU, via the installable `postgres` plugin (`loadr plugin install postgres`). "
                 "Same canvas, same runner, different protocol.",
        "src": "plugin-postgres.yaml",
        "notes": {
            "canvas": "The request nodes carry `postgres://` URLs instead of paths — the plugin resolves "
                      "the scheme at runtime.",
            "inspect-request": "The inspector shows the DSN and the parameterised query with its `params` — "
                               "a `status equals 1` check means the query succeeded.",
            "inspect-scenario": "`constant-arrival-rate` — a fixed queries-per-second rate against the DB, "
                                "open-model, like real traffic.",
            "run-live": "Direct SQL throughput streams like any other run — the desktop app treats a "
                        "protocol plugin exactly like HTTP.",
            "results": "Judged on DB-side gates: `postgres_req_duration` p95 and a checks rate — proving "
                       "plugins participate in thresholds too.",
        },
    },
    "observe-mixed": {
        "title": "observe-mixed — desktop walkthrough",
        "tagline": "Overlay system metrics on the load — correlation, not guesswork.",
        "intro": "**Load ↔ system, side by side.** loadr's `observe:` block pulls host and datastore metrics "
                 "(CPU, memory, load average, Postgres connections/commits, Redis ops) from Prometheus while "
                 "the load runs — so a latency bump lines up against the resource that caused it, instead of "
                 "you eyeballing two dashboards.",
        "src": "observe-mixed.yaml",
        "notes": {
            "canvas": "A `ramping-vus` scenario over a mixed read/write flow, with the `observe:` sources "
                      "attached at the plan level.",
            "inspect-scenario": "`ramping-vus` — ramp up, then hold, so the correlation has a steady window.",
            "run-live": "As VUs ramp, the system metrics move with them — that alignment is the whole point.",
            "results": "The summary carries the correlated system series alongside the load metrics; "
                       "`loadr report` renders the overlay.",
        },
    },
}

STEP_RE = re.compile(r"^\d+-(?P<slug>[a-z-]+)\.png$")


def build(demo: str) -> bool:
    d = DESKTOP / demo
    meta = NARRATIVES.get(demo)
    if meta is None:
        print(f"  ! no narrative for {demo}, skipping")
        return False
    shots = sorted(p.name for p in d.glob("*.png"))
    if not shots:
        print(f"  ! no screenshots in {d}, skipping")
        return False

    rows = []
    for name in shots:
        m = STEP_RE.match(name)
        slug = m.group("slug") if m else name
        desc = ACTION.get(slug, "")
        note = meta.get("notes", {}).get(slug)
        if note:
            desc = f"{desc} {note}" if desc else note
        rows.append(f'| <img src="{name}" width="{IMG_W}"> | {desc} |')

    body = [
        f"# {meta['title']}",
        "",
        meta["intro"],
        "",
        "| Screenshot | What's happening |",
        "|---|---|",
        *rows,
        "",
        "---",
        "",
        FOOTER.format(src=meta["src"]),
        "",
    ]
    (d / "README.md").write_text("\n".join(body))
    print(f"  ✓ {demo}/README.md  ({len(rows)} steps)")
    return True


def main() -> None:
    demos = sys.argv[1:] or list(NARRATIVES)
    for demo in demos:
        build(demo)


if __name__ == "__main__":
    main()
