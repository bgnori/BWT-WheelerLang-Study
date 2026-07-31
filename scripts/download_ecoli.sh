#!/bin/sh
# download_ecoli.sh
# Downloads the E. coli K-12 substr. MG1655 complete genome from NCBI in
# FASTA format and saves it to the data directory as ecoli.fna.
# The data file must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_ecoli.sh [data-dir]
#   data-dir defaults to ./data
#
# Source: NCBI RefSeq NC_000913.3 (E. coli K-12 substr. MG1655)
# Assembly: GCF_000005845.2 (ASM584v2)
# FTP URL:
#   https://ftp.ncbi.nlm.nih.gov/genomes/all/GCF/000/005/845/
#     GCF_000005845.2_ASM584v2/GCF_000005845.2_ASM584v2_genomic.fna.gz

set -e

DATADIR="${1:-./data}"
mkdir -p "$DATADIR"

FNA_URL="https://ftp.ncbi.nlm.nih.gov/genomes/all/GCF/000/005/845/GCF_000005845.2_ASM584v2/GCF_000005845.2_ASM584v2_genomic.fna.gz"
OUTFILE="$DATADIR/ecoli.fna"
TMPGZ="$DATADIR/_ecoli_tmp.fna.gz"

if [ -f "$OUTFILE" ]; then
    echo "FASTA data already exists: $OUTFILE"
    echo "Delete it first to re-download."
    exit 0
fi

echo "Downloading E. coli K-12 MG1655 genome from NCBI …"
if command -v wget > /dev/null 2>&1; then
    wget -q -O "$TMPGZ" "$FNA_URL"
elif command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "$TMPGZ" "$FNA_URL"
else
    echo "Error: neither wget nor curl found." >&2
    exit 1
fi

echo "Decompressing …"
gunzip -c "$TMPGZ" > "$OUTFILE"
rm -f "$TMPGZ"

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes, FASTA)"
