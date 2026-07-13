#!/usr/bin/env bash
# Re-capture the desktop walkthrough screenshots for one or more perf demos.
#
# Prereqs:
#   * the loadr repo checked out next to this one (../loadr with the desktop
#     app built: `cd desktop && npm ci && npm run build`) and a loadr binary
#     at target/debug/loadr
#   * the demo backend live: `make db && make api` (see the root README);
#     pass its address as BASE_URL
#   * for observe-mixed: the observe sidecars too (`make observe-up`)
#
#   BASE_URL=http://localhost:8080 ./capture.sh smoke journey ...
#   ./capture.sh            # all nine
set -uo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DESKTOP_DIR="$HERE/.."                                  # loadr-demos/desktop
PERF="$DESKTOP_DIR/../perf"
LOADR_REPO="${LOADR_REPO:-$DESKTOP_DIR/../../loadr}"    # the main loadr checkout
APP="$LOADR_REPO/desktop"
BASE_URL="${BASE_URL:-http://localhost:8080}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

demos=("$@")
[ ${#demos[@]} -eq 0 ] && demos=(smoke load stress spike soak impulse arrival-rate journey plugin-postgres observe-mixed)

for demo in "${demos[@]}"; do
  echo ">>> $demo"
  python3 "$APP/e2e/prep-demo-plan.py" "$PERF/$demo.yaml" "$WORK/$demo.yaml" \
    --cap 8 --base-url "$BASE_URL"
  rm -f "$HOME/.config/loadr-desktop/run-history.json"   # clean History panel
  out="$DESKTOP_DIR/$demo"
  rm -f "$out"/*.png
  (cd "$APP" && BASE_URL="$BASE_URL" LOADR_BIN="$LOADR_REPO/target/debug/loadr" \
    node e2e/demo-walkthroughs.mjs "$WORK/$demo.yaml" "$out" "$demo")
done

python3 "$HERE/build-readme.py" "${demos[@]}"
