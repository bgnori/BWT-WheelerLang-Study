#!/bin/sh
# Converts selected FASTA records to plain-text DNA suitable for FM-index
# indexing. The selector is an awk extended regular expression matched against
# FASTA header lines.
#
# Usage: ./scripts/prepare_fasta_records.sh [data-dir] [input-fasta-file] [output-text-file] [selector]
#   data-dir defaults to ./data
#   input-fasta-file defaults to osativa.fna
#   output-text-file defaults to osativa_chr1.txt
#   selector defaults to AP014957[.]1 (Oryza sativa chromosome 1)

set -e

DATADIR="${1:-./data}"
INPUT_FASTA="${2:-osativa.fna}"
OUTPUT_TEXT="${3:-osativa_chr1.txt}"
SELECTOR="${4:-AP014957[.]1}"
INFILE="$DATADIR/$INPUT_FASTA"
OUTFILE="$DATADIR/$OUTPUT_TEXT"
TMPFILE="$DATADIR/_${OUTPUT_TEXT}.tmp"

if [ ! -f "$INFILE" ]; then
    echo "Error: FASTA file not found: $INFILE" >&2
    echo "Download the FASTA file first." >&2
    exit 1
fi

if [ -f "$OUTFILE" ]; then
    echo "Processed data already exists: $OUTFILE"
    echo "Delete it first to re-process."
    exit 0
fi

echo "Processing FASTA records matching '$SELECTOR' ..."

if command -v awk > /dev/null 2>&1; then
    if ! awk -v selector="$SELECTOR" '
        /^>/ {
            if (active) {
                printf "\n"
                matched++
            }
            active = ($0 ~ selector)
            next
        }
        active { printf "%s", toupper($0) }
        END {
            if (active) {
                printf "\n"
                matched++
            }
            if (matched == 0) exit 2
        }
    ' "$INFILE" > "$TMPFILE"; then
        rm -f "$TMPFILE"
        echo "Error: no FASTA records matched selector: $SELECTOR" >&2
        exit 1
    fi
else
    echo "Error: awk not found." >&2
    exit 1
fi

mv "$TMPFILE" "$OUTFILE"
SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes, plain-text DNA)"