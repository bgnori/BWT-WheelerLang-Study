#!/bin/sh
# download_kaggle_amazon.sh
# Downloads Amazon-related datasets from Kaggle for benchmark tiers:
#   - small:  Amazon Laptop Prices Dataset
#   - medium: Amazon Mobile Dataset
#   - large:  Amazon Product Dataset (100K+)
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
        DATASET_NAME="Amazon Laptop Prices Dataset"
        DATASET_ID="${KAGGLE_DATASET_SMALL:-ionaskel/laptop-prices}"
        ;;
    medium)
        DATASET_NAME="Amazon Mobile Dataset"
        DATASET_ID="${KAGGLE_DATASET_MEDIUM:-PromptCloudHQ/amazon-unlocked-mobile}"
        ;;
    large)
        DATASET_NAME="Amazon Product Dataset (100K+)"
        DATASET_ID="${KAGGLE_DATASET_LARGE:-piyushjain16/amazon-product-dataset}"
        ;;
    *)
        echo "Error: invalid size '$SIZE' (use small|medium|large)." >&2
        exit 1
        ;;
esac

if ! command -v kaggle > /dev/null 2>&1; then
    echo "Error: kaggle command not found." >&2
    echo "Install Kaggle CLI first: pip install kaggle" >&2
    exit 1
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
