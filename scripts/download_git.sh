#!/bin/sh
# download_git.sh
# Downloads a specific release of the Git source code from GitHub, extracts
# the .c and .h files into data/git-src/, and removes everything else.
# The data directory must NOT be committed to the repository (see .gitignore).
#
# Usage:  ./scripts/download_git.sh [data-dir]
#   data-dir defaults to ./data

set -e

GIT_VERSION="2.45.0"
DATADIR="${1:-./data}"
OUTDIR="$DATADIR/git-src"
TMPTAR="$DATADIR/_git_tmp.tar.gz"
TMPEXT="$DATADIR/_git_tmp_extract"
TARBALL_URL="https://github.com/git/git/archive/refs/tags/v${GIT_VERSION}.tar.gz"

if [ -d "$OUTDIR" ] && [ "$(ls -A "$OUTDIR" 2>/dev/null)" ]; then
    echo "Git source already exists: $OUTDIR"
    echo "Delete it first to re-download."
    exit 0
fi

mkdir -p "$DATADIR"

echo "Downloading Git ${GIT_VERSION} source from GitHub …"
if command -v wget > /dev/null 2>&1; then
    wget -q -O "$TMPTAR" "$TARBALL_URL"
elif command -v curl > /dev/null 2>&1; then
    curl -fsSL -o "$TMPTAR" "$TARBALL_URL"
else
    echo "Error: neither wget nor curl found." >&2
    exit 1
fi

echo "Extracting archive …"
rm -rf "$TMPEXT"
mkdir -p "$TMPEXT"
tar -xzf "$TMPTAR" -C "$TMPEXT"

# The tarball unpacks to a single directory named git-<version>.
SRCROOT=$(find "$TMPEXT" -maxdepth 1 -type d -name "git-*" | head -1)
if [ -z "$SRCROOT" ]; then
    echo "Error: could not find extracted git source directory." >&2
    rm -rf "$TMPEXT" "$TMPTAR"
    exit 1
fi

echo "Copying .c and .h files to $OUTDIR …"
mkdir -p "$OUTDIR"
# Collect only top-level and direct sub-directory .c/.h files to keep the
# corpus manageable (git has hundreds of source files; all are welcome).
find "$SRCROOT" -name "*.c" -o -name "*.h" | while IFS= read -r f; do
    cp "$f" "$OUTDIR/"
done

rm -rf "$TMPEXT" "$TMPTAR"

COUNT=$(find "$OUTDIR" -type f | wc -l)
echo "Done: $OUTDIR (${COUNT} files)"
