#!/usr/bin/env bash
# Runs the sequential baseline and the parallel engine (1, 2, 4, ... workers)
# on the same input. Every run appends a line to data/runs.log.
set -euo pipefail

INPUT="${1:-data/measurements.csv}"
BIN="${BIN:-./bin/crunch}"
MAX_WORKERS="${MAX_WORKERS:-$(nproc 2>/dev/null || echo 4)}"

if [[ ! -f "$INPUT" ]]; then
  echo "input file '$INPUT' not found - run 'make data' first" >&2
  exit 1
fi

LOG="${LOG:-data/runs.log}"
BEFORE=$( [[ -f "$LOG" ]] && wc -l < "$LOG" || echo 0 )

echo "== sequential =="
"$BIN" -input "$INPUT" -mode sequential

for ((w = 1; w <= MAX_WORKERS; w *= 2)); do
  echo "== parallel, $w worker(s) =="
  "$BIN" -input "$INPUT" -mode parallel -workers "$w"
done

echo
echo "Recorded runs:"
tail -n +$((BEFORE + 1)) "$LOG"
