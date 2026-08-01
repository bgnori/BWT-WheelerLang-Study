#!/usr/bin/env python3
"""Summarize Phase A CSV output into simple ranking recommendations."""

from __future__ import annotations

import argparse
import csv
from pathlib import Path
from typing import Dict, List, Tuple


def load_rows(path: Path) -> List[Dict[str, str]]:
    with path.open(newline="", encoding="utf-8") as fh:
        return list(csv.DictReader(fh))


def summarize(path: Path) -> None:
    rows = load_rows(path)
    if not rows:
        print("no rows found")
        return
    print("Phase A summary")
    print("=" * 40)
    for dataset in sorted({row["dataset"] for row in rows}):
        ds_rows = [row for row in rows if row["dataset"] == dataset and row["search_mode"] == "warm"]
        if not ds_rows:
            continue
        print(f"[{dataset}]")
        for metric in ("search_p50_ms", "search_p95_ms", "search_peak_rss_kb", "index_bytes"):
            best = min(ds_rows, key=lambda r: float(r[metric]))
            print(f"  best {metric}: {best['occ']} ({best['backend']}) -> {best[metric]}")
        print()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Summarize Phase A benchmark CSV")
    parser.add_argument("csv", default="phase_a_results.csv")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    summarize(Path(args.csv))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
