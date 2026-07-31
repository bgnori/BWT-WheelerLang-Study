#!/bin/sh
# prepare_kaggle_amazon.sh
# Converts Kaggle Amazon dataset CSV/TSV files into a normalized plain-text file
# for FM-index benchmark input.
#
# Usage:
#   ./scripts/prepare_kaggle_amazon.sh <small|medium|large> [data-dir] [output-file]
#   data-dir defaults to ./data
#   output-file defaults to amazon_<size>.txt

set -e

SIZE="$1"
DATADIR="${2:-./data}"
OUTNAME="${3:-amazon_${SIZE}.txt}"

if [ -z "$SIZE" ]; then
    echo "Usage: $0 <small|medium|large> [data-dir] [output-file]" >&2
    exit 1
fi

case "$SIZE" in
    small|medium|large) ;;
    *)
        echo "Error: invalid size '$SIZE' (use small|medium|large)." >&2
        exit 1
        ;;
esac

if ! command -v python3 > /dev/null 2>&1; then
    echo "Error: python3 command not found." >&2
    exit 1
fi

EXTRACT_DIR="$DATADIR/kaggle/$SIZE/extracted"
OUTFILE="$DATADIR/$OUTNAME"

if [ ! -d "$EXTRACT_DIR" ]; then
    echo "Error: extracted dataset directory not found: $EXTRACT_DIR" >&2
    echo "Run download_kaggle_amazon.sh first." >&2
    exit 1
fi

if [ -f "$OUTFILE" ]; then
    echo "Processed data already exists: $OUTFILE"
    echo "Delete it first to re-process."
    exit 0
fi

echo "Preparing [$SIZE] Kaggle dataset into FM-index text: $OUTFILE"
python3 - "$EXTRACT_DIR" "$OUTFILE" <<'PY'
import csv
import os
import re
import sys
from pathlib import Path

in_dir = Path(sys.argv[1])
out_file = Path(sys.argv[2])

candidate_files = []
for p in in_dir.rglob("*"):
    if not p.is_file():
        continue
    lower = p.name.lower()
    if lower.endswith(".csv") or lower.endswith(".tsv"):
        candidate_files.append(p)

if not candidate_files:
    raise SystemExit(f"Error: no CSV/TSV files found in {in_dir}")

preferred_tokens = (
    "title", "name", "product", "brand", "description", "about",
    "category", "model", "feature", "spec", "review", "text",
)

def open_with_fallback(path):
    encodings = ("utf-8-sig", "utf-8", "cp932", "latin-1")
    for enc in encodings:
        try:
            f = open(path, "r", encoding=enc, newline="")
            f.read(4096)
            f.seek(0)
            return f
        except UnicodeDecodeError:
            continue
    return open(path, "r", encoding="utf-8", errors="replace", newline="")

def clean_text(s):
    s = re.sub(r"\s+", " ", s.strip())
    return s

rows_written = 0
files_used = 0
with open(out_file, "w", encoding="utf-8", newline="\n") as out:
    for path in sorted(candidate_files):
        delimiter = "\t" if path.suffix.lower() == ".tsv" else ","
        with open_with_fallback(path) as f:
            reader = csv.DictReader(f, delimiter=delimiter)
            if not reader.fieldnames:
                continue

            fields = [h for h in reader.fieldnames if h]
            lowered = [h.lower() for h in fields]
            selected = [
                fields[i] for i, lh in enumerate(lowered)
                if any(tok in lh for tok in preferred_tokens)
            ]
            if not selected:
                selected = fields

            file_rows = 0
            for row in reader:
                values = []
                for key in selected:
                    raw = row.get(key, "")
                    if raw is None:
                        continue
                    txt = clean_text(str(raw))
                    if txt:
                        values.append(txt)
                if not values:
                    continue
                out.write(" | ".join(values) + "\n")
                rows_written += 1
                file_rows += 1

            if file_rows > 0:
                files_used += 1

if rows_written == 0:
    raise SystemExit("Error: no valid text rows were produced from input files.")

print(f"Files used: {files_used}")
print(f"Rows written: {rows_written}")
PY

SIZE_BYTES=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE_BYTES} bytes)"
