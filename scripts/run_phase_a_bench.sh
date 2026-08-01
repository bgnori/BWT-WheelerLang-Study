#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

output_path="phase_a_results.csv"
stage="phase-a"

prev=""
for arg in "$@"; do
  if [[ "$prev" == "--output" ]]; then
    output_path="$arg"
  fi
  if [[ "$prev" == "--stage" ]]; then
    stage="$arg"
  fi
  prev="$arg"
done

python3 scripts/phase_a_bench.py "$@"

if [[ "${1:-}" == "--help" ]]; then
  exit 0
fi

if [[ "$stage" == "occ-precheck" ]]; then
  exit 0
fi

if [[ -f "$output_path" ]]; then
  python3 scripts/phase_a_summary.py "$output_path"
fi
