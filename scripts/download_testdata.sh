#!/bin/sh
# download_testdata.sh
# Downloads the Moby Dick text from Project Gutenberg into the data/ directory.
# The data file must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_testdata.sh [data-dir]
#   data-dir defaults to ./data

set -e

DATADIR="${1:-./data}"
mkdir -p "$DATADIR"

URL="https://www.gutenberg.org/cache/epub/2701/pg2701.txt"
OUTFILE="$DATADIR/moby_dick.txt"

if [ -f "$OUTFILE" ]; then
    echo "Test data already exists: $OUTFILE"
    echo "Delete it first to re-download."
    exit 0
fi

echo "Downloading Moby Dick (Project Gutenberg #2701) …"
if command -v wget > /dev/null 2>&1; then
    wget -q -O "$OUTFILE" "$URL"
elif command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "$OUTFILE" "$URL"
else
    echo "Error: neither wget nor curl found." >&2
    exit 1
fi

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes)"
