#!/bin/sh
# download_kenshin.sh
# Downloads Uesugi Kenshin (上杉謙信) from Aozora Bunko (青空文庫),
# extracts the Shift-JIS text, converts it to UTF-8, and saves it to the
# data directory as kenshin.txt.
# The data file must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_kenshin.sh [data-dir]
#   data-dir defaults to ./data
#
# Source page:
#   https://www.aozora.gr.jp/cards/001562/card56461.html
# Zip URL (ルビあり テキスト):
#   https://www.aozora.gr.jp/cards/001562/files/56461_ruby_53134.zip

set -e

DATADIR="${1:-./data}"
mkdir -p "$DATADIR"

ZIP_URL="https://www.aozora.gr.jp/cards/001562/files/56461_ruby_53134.zip"
OUTFILE="$DATADIR/kenshin.txt"
TMPZIP="$DATADIR/_kenshin_tmp.zip"
TMPDIR="$DATADIR/_kenshin_tmp_extract"

if [ -f "$OUTFILE" ]; then
    echo "Test data already exists: $OUTFILE"
    echo "Delete it first to re-download."
    exit 0
fi

echo "Downloading 上杉謙信 (Aozora Bunko card 56461) …"
if command -v wget > /dev/null 2>&1; then
    wget -q -O "$TMPZIP" "$ZIP_URL"
elif command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "$TMPZIP" "$ZIP_URL"
else
    echo "Error: neither wget nor curl found." >&2
    exit 1
fi

echo "Extracting zip …"
rm -rf "$TMPDIR"
mkdir -p "$TMPDIR"
unzip -q "$TMPZIP" -d "$TMPDIR"

# Aozora Bunko text files are encoded in Shift-JIS (CP932).
# Find the first .txt file in the extracted directory.
SJIS_FILE=$(find "$TMPDIR" -name "*.txt" | head -1)
if [ -z "$SJIS_FILE" ]; then
    echo "Error: no .txt file found in zip." >&2
    rm -rf "$TMPDIR" "$TMPZIP"
    exit 1
fi

echo "Converting Shift-JIS → UTF-8 …"
if command -v iconv > /dev/null 2>&1; then
    iconv -f CP932 -t UTF-8 "$SJIS_FILE" > "$OUTFILE"
elif command -v python3 > /dev/null 2>&1; then
    python3 -c "
import sys
with open(sys.argv[1], 'rb') as f:
    data = f.read()
with open(sys.argv[2], 'w', encoding='utf-8') as f:
    f.write(data.decode('cp932'))
" "$SJIS_FILE" "$OUTFILE"
else
    echo "Error: neither iconv nor python3 found for encoding conversion." >&2
    rm -rf "$TMPDIR" "$TMPZIP"
    exit 1
fi

rm -rf "$TMPDIR" "$TMPZIP"

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes, UTF-8)"
