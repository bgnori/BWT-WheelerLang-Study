#!/usr/bin/env python3
"""Summarize Phase A CSV output into simple ranking recommendations."""

from __future__ import annotations

import argparse
import csv
import statistics
from collections import defaultdict
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

        grouped: Dict[Tuple[str, str, str], List[Dict[str, str]]] = defaultdict(list)
        for row in ds_rows:
            key = (row["algo"], row["backend"], row["occ"])
            grouped[key].append(row)

        reduced: List[Dict[str, float]] = []
        for (algo, backend, occ), group in grouped.items():
            reduced.append(
                {
                    "algo": algo,
                    "backend": backend,
                    "occ": occ,
                    "search_p50_ms": float(statistics.median(float(r["search_p50_ms"]) for r in group)),
                    "search_p95_ms": float(statistics.median(float(r["search_p95_ms"]) for r in group)),
                    "search_peak_rss_kb": float(statistics.median(float(r["search_peak_rss_kb"]) for r in group)),
                    "index_bytes": float(statistics.median(float(r["index_bytes"]) for r in group)),
                }
            )

        for metric in ("search_p50_ms", "search_p95_ms", "search_peak_rss_kb", "index_bytes"):
            best = min(reduced, key=lambda r: r[metric])
            print(f"  best {metric}: {best['occ']} ({best['backend']}, {best['algo']}) -> {best[metric]:.3f}")
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
