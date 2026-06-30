# loadr-demos — local development helpers.
.PHONY: help db api test perf perf-all observe-up observe-down observe clean

DATABASE_URL ?= postgres://demo:demo@localhost:5432/storefront?sslmode=disable
BASE_URL     ?= http://localhost:8080
PLAN         ?= smoke

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-10s\033[0m %s\n",$$1,$$2}'

db: ## start local Postgres (docker compose)
	docker compose up -d

api: ## run the API against the local DB + Redis on :8080
	DATABASE_URL="$(DATABASE_URL)" REDIS_URL="redis://localhost:6379" ADDR=":8080" go run ./cmd/api

test: ## run unit tests
	go test ./... -race

perf: ## run one plan, e.g. make perf PLAN=journey   (needs the loadr CLI + a running API)
	BASE_URL="$(BASE_URL)" loadr run perf/$(PLAN).yaml --junit $(PLAN)-junit.xml --summary-export $(PLAN)-summary.json

perf-all: ## run every plan sequentially
	@for p in smoke load stress spike arrival-rate soak journey impulse; do \
		echo "== $$p =="; BASE_URL="$(BASE_URL)" loadr run perf/$$p.yaml || exit 1; \
	done

observe-up: ## start the observe sidecars (node/postgres/redis exporters + Prometheus); needs `make db` first
	docker compose -f observe/docker-compose.yml up -d

observe-down: ## stop the observe sidecars
	docker compose -f observe/docker-compose.yml down

observe: ## run the load↔system correlation demo and write observe.html (needs a loadr build with `observe:` + observe-up + api)
	BASE_URL="$(BASE_URL)" loadr run perf/observe-mixed.yaml --summary-export observe.json
	loadr report observe.json -o observe.html
	@echo "open observe.html"

clean: ## stop the DB + observe sidecars and remove build/report artifacts
	docker compose down -v
	docker compose -f observe/docker-compose.yml down 2>/dev/null || true
	rm -f loadr-demo-api *-junit.xml *-summary.json observe.json observe.html
