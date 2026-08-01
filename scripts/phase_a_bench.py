#!/usr/bin/env python3
"""Phase A benchmarking driver for the performance plan.

This script implements the core workflow described in docs/performance_plan_phase_a.md:
- build FM-index variants and stdlib suffixarray indexes
- run exact and regex searches
- collect build/search timings, RSS estimates, and CSV output

Usage examples:
  python3 scripts/phase_a_bench.py --stage screening
  python3 scripts/phase_a_bench.py --stage full --output data/phase_a_results.csv
"""

from __future__ import annotations

import argparse
import csv
import os
import shutil
import statistics
import subprocess
import sys
import time
from pathlib import Path
from typing import Dict, List, Tuple

REPO_ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = REPO_ROOT / "data"
OUTPUT_DIR = DATA_DIR / "phase_a_runs"
DEFAULT_OUTPUT = REPO_ROOT / "phase_a_results.csv"
BINARY = REPO_ROOT / "textindex"

DATASETS = {
    "kenshin": {
        "label": "Kenshin",
        "input": DATA_DIR / "kenshin.txt",
        "mode": "single",
        "queries": [
            ("exact_high", "謙信"),
            ("exact_low", "越前"),
            ("exact_miss", "織田信長公記"),
            ("regex_or", "上杉謙信|武田信玄"),
        ],
    },
    "mobydick": {
        "label": "MobyDick",
        "input": DATA_DIR / "moby_dick.txt",
        "mode": "single",
        "queries": [
            ("exact_high", "whale"),
            ("exact_low", "harpooner"),
            ("exact_miss", "microservice"),
            ("regex_or", "white|whale"),
        ],
    },
    "gitsource": {
        "label": "GitSource",
        "input": DATA_DIR / "git-src",
        "mode": "multi",
        "queries": [
            ("exact_high", "struct"),
            ("exact_low", "chdir-notify"),
            ("exact_miss", "kubernetes"),
            ("regex_or", "commit|diff"),
        ],
    },
    "ecoli": {
        "label": "Ecoli",
        "input": DATA_DIR / "ecoli.txt",
        "mode": "single",
        "queries": [
            ("exact_high", "AAAAAA"),
            ("exact_low", "ATGAAACGC"),
            ("exact_miss", "NNNNNNNN"),
            ("regex_or", "ATGAAACGC|GTTACCTGCC"),
        ],
    },
}


class BenchError(RuntimeError):
    pass


def ensure_binary() -> None:
    if not BINARY.exists():
        print("building textindex binary...")
        subprocess.run(["go", "build", "-o", str(BINARY), "./cmd/textindex"], cwd=REPO_ROOT, check=True)
    if not BINARY.exists():
        raise BenchError("textindex binary was not created")


def iter_input_files(dataset: str) -> List[Path]:
    info = DATASETS[dataset]
    if info["mode"] == "single":
        return [info["input"]]
    root = info["input"]
    files = sorted(root.rglob("*.c")) + sorted(root.rglob("*.h"))
    if not files:
        raise BenchError(f"no source files found under {root}")
    return files


def config_list(stage: str) -> List[Dict[str, str]]:
    core = [
        {"name": "rlbwt", "backend": "fm", "occ": "rlbwt", "algo": "sais"},
        {"name": "bitvectors", "backend": "fm", "occ": "bitvectors", "algo": "sais"},
        {"name": "waveletmatrix", "backend": "fm", "occ": "waveletmatrix", "algo": "sais"},
    ]
    new_occ = [
        {"name": "rrr", "backend": "fm", "occ": "rrr", "algo": "sais"},
        {"name": "eliasfano", "backend": "fm", "occ": "eliasfano", "algo": "sais"},
        {"name": "poppy", "backend": "fm", "occ": "poppy", "algo": "sais"},
        {"name": "dynamic", "backend": "fm", "occ": "dynamic", "algo": "sais"},
    ]
    if stage == "screening":
        return core + new_occ
    if stage == "expand":
        return new_occ
    if stage == "full":
        return core + new_occ + [{"name": "suffixarray", "backend": "suffixarray", "occ": "suffixarray", "algo": "suffixarray"}]
    raise BenchError(f"unknown stage: {stage}")


def maybe_skip_query(query_type: str, backend: str) -> bool:
    if query_type == "regex_or" and backend != "fm":
        return True
    return False


def read_rss_kb(pid: int) -> int:
    status_path = Path(f"/proc/{pid}/status")
    if not status_path.exists():
        return 0
    try:
        for line in status_path.read_text(encoding="utf-8", errors="ignore").splitlines():
            if line.startswith("VmRSS:"):
                return int(line.split()[1])
    except FileNotFoundError:
        return 0
    return 0


def run_process(cmd: List[str], cwd: Path) -> Tuple[int, str, int]:
    start = time.perf_counter()
    proc = subprocess.Popen(cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    peak_rss_kb = 0
    while True:
        if proc.poll() is not None:
            break
        peak_rss_kb = max(peak_rss_kb, read_rss_kb(proc.pid))
        time.sleep(0.1)
    if proc.stdout is not None:
        output = proc.stdout.read()
    else:
        output = ""
    elapsed_sec = time.perf_counter() - start
    if proc.returncode != 0:
        raise BenchError(f"command failed ({proc.returncode}): {' '.join(cmd)}\n{output}")
    return proc.returncode, output, int(max(peak_rss_kb, read_rss_kb(proc.pid)))


def run_build(dataset: str, config: Dict[str, str], output_path: Path) -> Tuple[float, int, int]:
    info = DATASETS[dataset]
    input_path = info["input"]
    if info["mode"] == "single":
        if config["backend"] == "suffixarray":
            cmd = [str(BINARY), "build", "--algo", config["algo"], str(input_path), str(output_path)]
        else:
            cmd = [str(BINARY), "build", "--algo", config["algo"], "--occ", config["occ"], str(input_path), str(output_path)]
    else:
        files = iter_input_files(dataset)
        if config["backend"] == "suffixarray":
            cmd = [str(BINARY), "build-multi", "--algo", config["algo"], str(output_path)] + [str(p) for p in files]
        else:
            cmd = [str(BINARY), "build-multi", "--algo", config["algo"], "--occ", config["occ"], str(output_path)] + [str(p) for p in files]

    _, _, peak_rss_kb = run_process(cmd, REPO_ROOT)
    elapsed = 0.0
    # measure elapsed separately with a fresh run to ensure timing is recorded
    start = time.perf_counter()
    proc = subprocess.Popen(cmd, cwd=REPO_ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
    peak = 0
    while True:
        if proc.poll() is not None:
            break
        peak = max(peak, read_rss_kb(proc.pid))
        time.sleep(0.1)
    output = proc.stdout.read() if proc.stdout is not None else ""
    elapsed = time.perf_counter() - start
    if proc.returncode != 0:
        raise BenchError(f"build failed: {' '.join(cmd)}\n{output}")
    return elapsed, peak, int(os.path.getsize(output_path) if output_path.exists() else 0)


def run_search(dataset: str, config: Dict[str, str], index_path: Path, query_type: str, query: str, run_count: int) -> Tuple[List[float], int, int]:
    if maybe_skip_query(query_type, config["backend"]):
        return [], 0, 0
    cmd = [str(BINARY), "search", "--limit", "1000", str(index_path), query]
    per_run_ms: List[float] = []
    peak_rss_values: List[int] = []
    hits_values: List[int] = []
    for i in range(run_count):
        start = time.perf_counter()
        proc = subprocess.Popen(cmd, cwd=REPO_ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        peak = 0
        while True:
            if proc.poll() is not None:
                break
            peak = max(peak, read_rss_kb(proc.pid))
            time.sleep(0.05)
        output = proc.stdout.read() if proc.stdout is not None else ""
        elapsed_ms = (time.perf_counter() - start) * 1000.0
        if proc.returncode != 0:
            raise BenchError(f"search failed: {' '.join(cmd)}\n{output}")
        hit_count = len([line for line in output.splitlines() if line.strip() and not line.startswith("warning:")])
        per_run_ms.append(elapsed_ms)
        peak_rss_values.append(peak)
        hits_values.append(hit_count)
    return per_run_ms, int(statistics.median(peak_rss_values)), int(statistics.median(hits_values))


def preflight_query(dataset: str, query_type: str, query: str, config: Dict[str, str]) -> None:
    if query_type != "exact_miss":
        return
    info = DATASETS[dataset]
    index_path = OUTPUT_DIR / f"{dataset}_{config['name']}.idx"
    if not index_path.exists():
        raise BenchError(f"index missing for preflight: {index_path}")
    cmd = [str(BINARY), "search", "--limit", "100", str(index_path), query]
    proc = subprocess.run(cmd, cwd=REPO_ROOT, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True, check=False)
    output = proc.stdout or ""
    hit_count = len([line for line in output.splitlines() if line.strip() and not line.startswith("warning:")])
    if hit_count != 0:
        raise BenchError(f"zero-hit query unexpectedly matched: dataset={dataset} query={query} hits={hit_count}")


def benchmark_dataset(dataset: str, stage: str, output_rows: List[Dict[str, object]], build_repeats: int) -> None:
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    info = DATASETS[dataset]
    for config in config_list(stage):
        index_path = OUTPUT_DIR / f"{dataset}_{config['name']}.idx"
        if config["backend"] == "suffixarray" and stage != "full":
            continue
        if config["backend"] != "suffixarray" and not index_path.exists():
            pass
        # build repeated
        build_elapsed_values = []
        build_peak_values = []
        build_size_values = []
        for _ in range(build_repeats):
            if index_path.exists():
                index_path.unlink()
            elapsed, peak_rss_kb, index_bytes = run_build(dataset, config, index_path)
            build_elapsed_values.append(elapsed)
            build_peak_values.append(peak_rss_kb)
            build_size_values.append(index_bytes)
        build_elapsed_sec = round(statistics.median(build_elapsed_values), 6)
        build_peak_rss_kb = int(statistics.median(build_peak_values))
        index_bytes = int(statistics.median(build_size_values))

        for query_type, query in info["queries"]:
            if maybe_skip_query(query_type, config["backend"]):
                continue
            preflight_query(dataset, query_type, query, config)
            per_run_ms, search_peak_rss_kb, hits = run_search(dataset, config, index_path, query_type, query, run_count=10)
            if not per_run_ms:
                continue
            search_p50_ms = round(statistics.median(per_run_ms), 3)
            search_p95_ms = round(statistics.quantiles(per_run_ms, n=20)[-1], 3) if len(per_run_ms) >= 20 else round(max(per_run_ms), 3)
            search_elapsed_ms = round(per_run_ms[0], 3)
            row = {
                "dataset": dataset,
                "dataset_bytes": int(info["input"].stat().st_size) if info["input"].exists() else 0,
                "algo": config["algo"],
                "occ": config["occ"],
                "backend": config["backend"],
                "query_type": query_type,
                "query": query,
                "search_mode": "cold",
                "build_elapsed_sec": build_elapsed_sec,
                "build_peak_rss_kb": build_peak_rss_kb,
                "index_bytes": index_bytes,
                "search_elapsed_ms": search_elapsed_ms,
                "search_p50_ms": search_p50_ms,
                "search_p95_ms": search_p95_ms,
                "search_peak_rss_kb": search_peak_rss_kb,
                "hits": hits,
                "run_id": f"{dataset}-{config['name']}-{query_type}",
            }
            output_rows.append(row)

            # warm measurements: use the 2nd..10th runs as warm sample
            warm_runs = per_run_ms[1:]
            if warm_runs:
                warm_p50 = round(statistics.median(warm_runs), 3)
                warm_p95 = round(statistics.quantiles(warm_runs, n=20)[-1], 3) if len(warm_runs) >= 20 else round(max(warm_runs), 3)
                warm_elapsed = round(warm_runs[0], 3)
                warm_row = dict(row)
                warm_row["search_mode"] = "warm"
                warm_row["search_elapsed_ms"] = warm_elapsed
                warm_row["search_p50_ms"] = warm_p50
                warm_row["search_p95_ms"] = warm_p95
                warm_row["run_id"] = f"{dataset}-{config['name']}-{query_type}-warm"
                output_rows.append(warm_row)


def write_csv(rows: List[Dict[str, object]], output_path: Path) -> None:
    fieldnames = [
        "dataset",
        "dataset_bytes",
        "algo",
        "occ",
        "backend",
        "query_type",
        "query",
        "search_mode",
        "build_elapsed_sec",
        "build_peak_rss_kb",
        "index_bytes",
        "search_elapsed_ms",
        "search_p50_ms",
        "search_p95_ms",
        "search_peak_rss_kb",
        "hits",
        "run_id",
    ]
    with output_path.open("w", newline="", encoding="utf-8") as fh:
        writer = csv.DictWriter(fh, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run the Phase A benchmark workflow")
    parser.add_argument("--stage", choices=["screening", "expand", "full"], default="screening")
    parser.add_argument("--output", default=str(DEFAULT_OUTPUT))
    parser.add_argument("--build-repeats", type=int, default=3)
    parser.add_argument("--datasets", nargs="*", choices=list(DATASETS.keys()), default=list(DATASETS.keys()))
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    ensure_binary()
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    rows: List[Dict[str, object]] = []
    for dataset in args.datasets:
        benchmark_dataset(dataset, args.stage, rows, build_repeats=args.build_repeats)
    output_path = Path(args.output)
    output_path.parent.mkdir(parents=True, exist_ok=True)
    write_csv(rows, output_path)
    print(f"wrote {len(rows)} rows to {output_path}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BenchError as exc:
        print(f"error: {exc}", file=sys.stderr)
        raise SystemExit(2)
