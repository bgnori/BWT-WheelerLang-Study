#!/bin/sh
# generate_fake_logs.sh
# Generates fake log test data using flog and mclogs.
#
# Usage:
#   ./scripts/generate_fake_logs.sh <apache-common|apache-error|syslog|json|logfmt> [size] [data-dir]
#   size defaults to 1M
#   data-dir defaults to ./data
#
# Examples:
#   ./scripts/generate_fake_logs.sh apache-common 10M
#   ./scripts/generate_fake_logs.sh json 512K ./data

set -e

KIND="$1"
SIZE_RAW="${2:-1M}"
DATADIR="${3:-./data}"
OUTDIR="$DATADIR/fake-logs"

if [ -z "$KIND" ]; then
    echo "Usage: $0 <apache-common|apache-error|syslog|json|logfmt> [size] [data-dir]" >&2
    exit 1
fi

case "$SIZE_RAW" in
    *[!0-9KkMmGg]*|"")
        echo "Error: invalid size '$SIZE_RAW' (examples: 1048576, 512K, 1M, 2G)." >&2
        exit 1
        ;;
esac

case "$SIZE_RAW" in
    *[Kk])
        NUM=${SIZE_RAW%[Kk]}
        BYTES=$((NUM * 1024))
        ;;
    *[Mm])
        NUM=${SIZE_RAW%[Mm]}
        BYTES=$((NUM * 1024 * 1024))
        ;;
    *[Gg])
        NUM=${SIZE_RAW%[Gg]}
        BYTES=$((NUM * 1024 * 1024 * 1024))
        ;;
    *)
        BYTES=$SIZE_RAW
        ;;
esac

if [ -z "$BYTES" ] || [ "$BYTES" -le 0 ] 2>/dev/null; then
    echo "Error: size must be a positive integer." >&2
    exit 1
fi

mkdir -p "$OUTDIR"

generate_with_flog() {
    FORMAT="$1"
    OUTFILE="$2"
    if ! command -v flog > /dev/null 2>&1; then
        echo "Error: flog command not found. Install: https://github.com/mingrammer/flog" >&2
        exit 1
    fi
    flog -f "$FORMAT" -b "$BYTES" -o "$OUTFILE" -t log
}

generate_with_mclogs() {
    FORMAT="$1"
    OUTFILE="$2"
    if ! command -v mclogs > /dev/null 2>&1; then
        echo "Error: mclogs command not found. Install: https://github.com/joanlopez/mclogs" >&2
        exit 1
    fi
    mclogs --format "$FORMAT" --bytes "$BYTES" --output "$OUTFILE"
}

case "$KIND" in
    apache-common)
        OUTFILE="$OUTDIR/flog_apache_common.log"
        generate_with_flog "apache_common" "$OUTFILE"
        ;;
    apache-error)
        OUTFILE="$OUTDIR/flog_apache_error.log"
        generate_with_flog "apache_error" "$OUTFILE"
        ;;
    syslog)
        OUTFILE="$OUTDIR/flog_syslog.log"
        generate_with_flog "rfc3164" "$OUTFILE"
        ;;
    json)
        OUTFILE="$OUTDIR/mclogs_json.log"
        generate_with_mclogs "json" "$OUTFILE"
        ;;
    logfmt)
        OUTFILE="$OUTDIR/mclogs_logfmt.log"
        generate_with_mclogs "logfmt" "$OUTFILE"
        ;;
    *)
        echo "Error: unknown kind '$KIND'." >&2
        echo "Use one of: apache-common, apache-error, syslog, json, logfmt" >&2
        exit 1
        ;;
esac

SIZE=$(wc -c < "$OUTFILE")
echo "Done: $OUTFILE (${SIZE} bytes)"
