#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

python3 scripts/phase_a_bench.py "$@"

if [[ "${1:-}" == "--help" ]]; then
  exit 0
fi

if [[ -f phase_a_results.csv ]]; then
  python3 scripts/phase_a_summary.py phase_a_results.csv
fi
