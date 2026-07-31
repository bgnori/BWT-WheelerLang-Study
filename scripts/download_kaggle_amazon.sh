#!/bin/sh
# download_kaggle_amazon.sh
# Downloads Amazon-related community datasets from Kaggle for benchmark tiers.
# Kaggle does not provide an official Amazon product dataset source here. The
# small tier defaults to owm4096/laptop-prices; set KAGGLE_DATASET_SMALL,
# KAGGLE_DATASET_MEDIUM, or KAGGLE_DATASET_LARGE to choose a different source.
# The downloaded files must NOT be committed to the repository.
#
# Usage:
#   ./scripts/download_kaggle_amazon.sh <small|medium|large> [data-dir]
#   data-dir defaults to ./data
#
# Requirements:
# - kaggle CLI installed and authenticated (KAGGLE_USERNAME/KAGGLE_KEY or ~/.kaggle/kaggle.json)

set -e

SIZE="$1"
DATADIR="${2:-./data}"

if [ -z "$SIZE" ]; then
    echo "Usage: $0 <small|medium|large> [data-dir]" >&2
    exit 1
fi

case "$SIZE" in
    small)
        DATASET_NAME="Laptop Prices"
        DATASET_ID="${KAGGLE_DATASET_SMALL:-owm4096/laptop-prices}"
        ;;
    medium)
        DATASET_NAME="Amazon medium dataset"
        DATASET_ID="${KAGGLE_DATASET_MEDIUM:-}"
        ;;
    large)
        DATASET_NAME="Amazon large dataset"
        DATASET_ID="${KAGGLE_DATASET_LARGE:-}"
        ;;
    *)
        echo "Error: invalid size '$SIZE' (use small|medium|large)." >&2
        exit 1
        ;;
esac

if [ -z "$DATASET_ID" ]; then
    ENV_NAME="KAGGLE_DATASET_$(printf '%s' "$SIZE" | tr '[:lower:]' '[:upper:]')"
    echo "Error: no Kaggle dataset ID configured for size '$SIZE'." >&2
    echo "Kaggle Amazon datasets are community-published; choose a source you trust and set $ENV_NAME." >&2
    echo "Example: $ENV_NAME=owner/dataset-slug make download-amazon-$SIZE" >&2
    exit 1
fi

if ! command -v kaggle > /dev/null 2>&1; then
    echo "Error: kaggle command not found." >&2
    echo "Install Kaggle CLI first: pip install kaggle" >&2
    exit 1
fi

if [ -z "${KAGGLE_USERNAME:-}" ] || [ -z "${KAGGLE_KEY:-}" ]; then
    if [ ! -f "$HOME/.kaggle/kaggle.json" ] && [ ! -f "$HOME/.kaggle/access_token" ]; then
        echo "Error: Kaggle credentials not found." >&2
        echo "Set KAGGLE_USERNAME and KAGGLE_KEY, place kaggle.json at ~/.kaggle/kaggle.json," >&2
        echo "or authenticate the Kaggle CLI so ~/.kaggle/access_token exists." >&2
        echo "If the dataset requires terms acceptance, accept it on Kaggle first." >&2
        exit 1
    fi
fi

RAW_DIR="$DATADIR/kaggle/$SIZE/raw"
EXTRACT_DIR="$DATADIR/kaggle/$SIZE/extracted"
mkdir -p "$RAW_DIR" "$EXTRACT_DIR"

echo "Downloading [$SIZE] $DATASET_NAME from Kaggle dataset: $DATASET_ID"
kaggle datasets download -d "$DATASET_ID" -p "$RAW_DIR" --force

ZIP_COUNT=0
for z in "$RAW_DIR"/*.zip; do
    if [ -f "$z" ]; then
        ZIP_COUNT=$((ZIP_COUNT + 1))
        unzip -o -q "$z" -d "$EXTRACT_DIR"
    fi
done

if [ "$ZIP_COUNT" -eq 0 ]; then
    echo "Warning: no zip files found in $RAW_DIR (dataset may contain direct files)." >&2
fi

FILE_COUNT=$(find "$EXTRACT_DIR" -type f | wc -l | tr -d ' ')
echo "Done: extracted files in $EXTRACT_DIR (${FILE_COUNT} files)"
