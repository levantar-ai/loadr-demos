# loadr-demos

A complete, runnable example of performance testing a real service with
[**loadr**](https://github.com/levantar-ai/loadr) inside GitHub Actions.

It ships:

- a small **Go + Postgres** storefront API (routes, a database, auth, an order
  transaction, plus deliberate CPU-bound and slow endpoints), and
- a **GitHub Actions** pipeline that builds the API once and then fans out into
  **seven performance tests running in parallel** — each on its own runner with
  its own throwaway Postgres — surfacing JUnit results in the Checks tab and a
  combined table on the run summary.

It's meant to be read, copied, and adapted.

---

## The pipeline

```
        ┌─────────┐
        │  build  │   compile the API once, share the binary
        └────┬────┘
             │
   ┌─────────┼───────────────────────────────────────────────┐
   ▼         ▼          ▼         ▼          ▼        ▼        ▼
 smoke     load      stress    spike   arrival-rate soak   journey     ← run in parallel,
   │         │          │         │          │        │        │         each with its own
   └─────────┴──────────┴────┬────┴──────────┴────────┴────────┘         Postgres service
                             ▼
                        ┌─────────┐
                        │ report  │   one table, all results, on the run summary
                        └─────────┘
```

Each perf job:

1. spins up a `postgres:16` **service container**,
2. starts the API (it runs migrations + seeds the catalog on boot),
3. installs and runs loadr via the first-party action:

   ```yaml
   - uses: levantar-ai/loadr@v1
     with:
       plan: perf/${{ matrix.test }}.yaml
       version: latest
       junit: ${{ matrix.test }}-junit.xml
       summary: ${{ matrix.test }}-summary.json
   ```
4. publishes the JUnit report to the **Checks** tab (via `dorny/test-reporter`),
5. uploads the JSON summary for the aggregate `report` job.

A breached [threshold](https://github.com/levantar-ai/loadr) fails the job
(loadr exits non-zero), so performance regressions break the build like any
other test. `fail-fast: false` means one failing test type never cancels the
rest.

See [`.github/workflows/perf.yml`](.github/workflows/perf.yml).

## The test types

Every plan lives in [`perf/`](perf/) and targets a different executor / failure
mode — together they cover the load-testing taxonomy:

| Plan | Executor | What it proves |
|------|----------|----------------|
| [`smoke`](perf/smoke.yaml) | `constant-vus` (2 VUs, 15s) | fast sanity gate, strict checks |
| [`load`](perf/load.yaml) | `constant-vus` (25 VUs) | steady expected traffic, p95/p99 budgets |
| [`stress`](perf/stress.yaml) | `ramping-vus` (0→80) | find the knee on the CPU-bound path; `abort_on_fail` |
| [`spike`](perf/spike.yaml) | `ramping-arrival-rate` (20→200/s) | sudden order surge + recovery (open model) |
| [`arrival-rate`](perf/arrival-rate.yaml) | `constant-arrival-rate` (150/s) | throughput & saturation (`dropped_iterations`) |
| [`soak`](perf/soak.yaml) | `constant-vus` (10, 2m) | latency drift / leaks under sustained load |
| [`journey`](perf/journey.yaml) | `constant-vus` (8) | login → extract token → authed write → order → verify, with a CSV **data feeder** + **correlation** |

These show off loadr features you'll actually use: closed vs open models,
ramping stages, think time, tag-filtered thresholds
(`http_req_duration{name:detail}`), abort-on-fail, CSV feeders, response
extraction (`jsonpath`/header) and request correlation, and JSON-body checks.

## The API

A storefront on Go's standard-library router + Postgres (`pgx`).

| Method & route | Description |
|---|---|
| `GET /healthz` / `GET /readyz` | liveness / readiness (readiness pings the DB) |
| `GET /api/products?q=&limit=&offset=` | list/search the catalog (reads DB) |
| `GET /api/products/{id}` | product detail |
| `POST /api/products` | create a product — **requires `Authorization: Bearer …`** |
| `POST /api/orders` | place an order — priced + stock-decremented in **one transaction** |
| `GET /api/orders/{id}` | fetch an order + its line items |
| `POST /api/auth/login` | returns a bearer token (demo auth) |
| `GET /api/compute?n=` | CPU-bound (SHA-256 ×n) — latency rises with load |
| `GET /api/slow?ms=` | controllable latency |

Source: [`cmd/api`](cmd/api) (entrypoint), [`internal/server`](internal/server)
(routes/handlers), [`internal/store`](internal/store) (Postgres, migrations,
seed).

## Run it locally

Prereqs: Go 1.24+, Docker, and the [loadr CLI](https://github.com/levantar-ai/loadr#install).

```bash
make db                       # start Postgres in Docker
make api                      # run the API on :8080 (migrates + seeds on boot)

# in another shell — run a plan against it:
make perf PLAN=smoke          # or: load, stress, spike, arrival-rate, soak, journey
make perf-all                 # run them all sequentially

make test                     # go unit tests
make clean                    # stop the DB, remove artifacts
```

Or drive loadr directly:

```bash
BASE_URL=http://localhost:8080 loadr run perf/journey.yaml \
  --junit journey-junit.xml --summary-export journey-summary.json
```

## License

MIT — see [LICENSE](LICENSE). Built to demonstrate
[loadr](https://github.com/levantar-ai/loadr).
