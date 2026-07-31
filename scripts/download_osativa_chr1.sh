#!/bin/sh
# Backward-compatible wrapper. Use download_osativa.sh for the full-genome
# Oryza sativa download.

set -e

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
exec "$SCRIPT_DIR/download_osativa.sh" "$@"
