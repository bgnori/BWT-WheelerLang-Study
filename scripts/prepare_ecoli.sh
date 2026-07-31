#!/bin/sh
# prepare_ecoli.sh
# Converts a FASTA file to a plain-text DNA sequence suitable for FM-index
# indexing. Strips header lines (lines starting with '>'), joins the sequence
# lines within each record into one continuous line, and uppercases all bases.
#
# Usage:  ./scripts/prepare_ecoli.sh [data-dir] [input-fasta-file] [output-text-file]
#   data-dir defaults to ./data
#   input-fasta-file defaults to ecoli.fna
#   output-text-file defaults to ecoli.txt
#
# Input:  <data-dir>/<input-fasta-file>  (FASTA)
# Output: <data-dir>/<output-text-file>  (plain-text DNA sequence, uppercase)
#
# Notes:
# - FASTA header lines ('>...') are removed; only sequence data is kept.
# - Bases are uppercased (a→A, c→C, g→G, t→T, n→N).
# - Each FASTA record (contig/chromosome) is written as a single line
#   followed by a newline, so FM-index search patterns do not span artificial
#   line breaks inserted by the reference assembly.
# - The FM-index uses 0x00 as a sentinel; normal FASTA sequences do not
#   contain null bytes, so no additional filtering is required.

set -e

DATADIR="${1:-./data}"
INPUT_FASTA="${2:-ecoli.fna}"
OUTPUT_TEXT="${3:-ecoli.txt}"
INFILE="$DATADIR/$INPUT_FASTA"
OUTFILE="$DATADIR/$OUTPUT_TEXT"

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

echo "Processing FASTA → plain text DNA …"

if command -v awk > /dev/null 2>&1; then
    # Strip header lines and join sequence lines for each FASTA record.
    # Each contig/chromosome becomes one uppercase line.
    awk '
        /^>/ {
            if (seq != "") { print toupper(seq); seq = "" }
            next
        }
        { seq = seq $0 }
        END { if (seq != "") print toupper(seq) }
    ' "$INFILE" > "$OUTFILE"
elif command -v python3 > /dev/null 2>&1; then
    python3 -c "
import sys
in_path, out_path = sys.argv[1], sys.argv[2]
with open(in_path) as fh, open(out_path, 'w') as out:
    seq = []
    for line in fh:
        line = line.rstrip('\n')
        if line.startswith('>'):
            if seq:
                out.write(''.join(seq).upper() + '\n')
                seq = []
        else:
            seq.append(line)
    if seq:
        out.write(''.join(seq).upper() + '\n')
" "$INFILE" "$OUTFILE"
else
    echo "Error: neither awk nor python3 found." >&2
    exit 1
fi

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes, plain-text DNA)"
