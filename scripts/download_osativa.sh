#!/bin/sh
# Downloads the Oryza sativa (rice) genome FASTA from NCBI and saves it as
# data/osativa.fna. Downstream prepare targets select the chromosome or contig
# to index from this full-genome FASTA.
# The data file must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_osativa.sh [data-dir]
#   data-dir defaults to ./data
#
# Source:
#   NCBI GenBank Assembly GCA_001433935.1 (IRGSP-1.0)
#   Genome FASTA:
#   https://ftp.ncbi.nlm.nih.gov/genomes/all/GCA/001/433/935/
#     GCA_001433935.1_IRGSP-1.0/
#     GCA_001433935.1_IRGSP-1.0_genomic.fna.gz

set -e

DATADIR="${1:-./data}"
mkdir -p "$DATADIR"

FNA_URL="https://ftp.ncbi.nlm.nih.gov/genomes/all/GCA/001/433/935/GCA_001433935.1_IRGSP-1.0/GCA_001433935.1_IRGSP-1.0_genomic.fna.gz"
OUTFILE="$DATADIR/osativa.fna"
TMPGZ="$DATADIR/_osativa_tmp.fna.gz"

if [ -f "$OUTFILE" ]; then
    echo "FASTA data already exists: $OUTFILE"
    echo "Delete it first to re-download."
    exit 0
fi

echo "Downloading Oryza sativa genome FASTA from NCBI ..."
if command -v wget > /dev/null 2>&1; then
    wget -q -O "$TMPGZ" "$FNA_URL"
elif command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "$TMPGZ" "$FNA_URL"
else
    echo "Error: neither wget nor curl found." >&2
    exit 1
fi

echo "Decompressing ..."
gunzip -c "$TMPGZ" > "$OUTFILE"
rm -f "$TMPGZ"

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes, FASTA)"