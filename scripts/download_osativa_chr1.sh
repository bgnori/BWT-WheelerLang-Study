#!/bin/sh
# download_osativa_chr1.sh
# Downloads Oryza sativa (rice) chromosome 1 genome FASTA from NCBI and saves
# it as data/osativa_chr1.fna.
# The data file must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_osativa_chr1.sh [data-dir]
#   data-dir defaults to ./data
#
# Source:
#   NCBI GenBank Assembly GCA_001433935.1 (IRGSP-1.0)
#   Chromosome 1 FASTA (~43.3MB plain FASTA):
#   https://ftp.ncbi.nlm.nih.gov/genomes/all/GCA/001/433/935/
#     GCA_001433935.1_IRGSP-1.0/
#     GCA_001433935.1_IRGSP-1.0_assembly_structure/
#     Primary_Assembly/assembled_chromosomes/FASTA/chr1.fa.gz

set -e

DATADIR="${1:-./data}"
mkdir -p "$DATADIR"

FNA_URL="https://ftp.ncbi.nlm.nih.gov/genomes/all/GCA/001/433/935/GCA_001433935.1_IRGSP-1.0/GCA_001433935.1_IRGSP-1.0_assembly_structure/Primary_Assembly/assembled_chromosomes/FASTA/chr1.fa.gz"
OUTFILE="$DATADIR/osativa_chr1.fna"
TMPGZ="$DATADIR/_osativa_chr1_tmp.fna.gz"

if [ -f "$OUTFILE" ]; then
    echo "FASTA data already exists: $OUTFILE"
    echo "Delete it first to re-download."
    exit 0
fi

echo "Downloading Oryza sativa chromosome 1 FASTA from NCBI …"
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
