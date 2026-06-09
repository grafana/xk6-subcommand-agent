#!/usr/bin/env bash
# Runs all functional k6 scripts (protocol + browser per workflow) and reports
# a summary. Exit code 0 if every script passed; non-zero (= number of failures)
# otherwise.
#
# Usage:
#   ./tests/run-all.sh
#   BASE_URL=http://localhost:3333 ./tests/run-all.sh
#
# Per-test logs at: ${TMPDIR:-/tmp}/k6-run-all/<workflow>-<kind>.log
#
# Auto-discovers workflows by listing tests/w*-*/ folders and matching the
# protocol.js + browser.js convention. Skips any folder that doesn't have both.

set -u

LOGDIR="${TMPDIR:-/tmp}/k6-run-all"
mkdir -p "$LOGDIR"

# Find all workflow folders matching tests/w<N>-<name>/
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKFLOWS=()
for d in "$SCRIPT_DIR"/w*-*/; do
  [ -d "$d" ] || continue
  [ -f "$d/protocol.js" ] || continue
  [ -f "$d/browser.js" ] || continue
  WORKFLOWS+=("$(basename "$d")")
done

if [ "${#WORKFLOWS[@]}" -eq 0 ]; then
  echo "No workflow folders found in $SCRIPT_DIR (expected tests/w<N>-<name>/protocol.js + browser.js)." >&2
  exit 0
fi

# Build the test list: protocol then browser per workflow.
declare -a RESULTS
FAIL_COUNT=0

for wf in "${WORKFLOWS[@]}"; do
  for kind in protocol browser; do
    script="tests/$wf/$kind.js"
    log="$LOGDIR/$wf-$kind.log"

    printf '%-20s %-50s ' "$wf $kind" "$script"
    start_ns=$(date +%s%N 2>/dev/null || python3 -c 'import time; print(int(time.time()*1e9))')

    if [ "$kind" = "browser" ]; then
      K6_BROWSER_HEADLESS=true k6 run "$script" >"$log" 2>&1
      rc=$?
    else
      k6 run "$script" >"$log" 2>&1
      rc=$?
    fi

    end_ns=$(date +%s%N 2>/dev/null || python3 -c 'import time; print(int(time.time()*1e9))')
    dur_s=$(awk "BEGIN { printf \"%.1f\", ($end_ns - $start_ns) / 1e9 }")

    if [ $rc -eq 0 ]; then
      printf 'OK   (%ss)\n' "$dur_s"
      RESULTS+=("OK|$wf $kind|${dur_s}s")
    else
      printf 'FAIL (%ss)  log: %s\n' "$dur_s" "$log"
      RESULTS+=("FAIL|$wf $kind|${dur_s}s|$log")
      FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
  done
done

echo
echo "--- summary ---"
for r in "${RESULTS[@]}"; do
  IFS='|' read -r status label dur extra <<<"$r"
  if [ "$status" = "OK" ]; then
    printf '  OK   %-30s %s\n' "$label" "$dur"
  else
    printf '  FAIL %-30s %s  (see %s)\n' "$label" "$dur" "$extra"
  fi
done
echo
TOTAL=$(( ${#WORKFLOWS[@]} * 2 ))
if [ $FAIL_COUNT -eq 0 ]; then
  echo "All $TOTAL functional tests passed."
else
  echo "$FAIL_COUNT of $TOTAL failed."
fi
exit $FAIL_COUNT
