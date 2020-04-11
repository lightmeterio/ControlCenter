#!/bin/bash

set -e
set -o pipefail

UTILS_DIR="$(dirname $(realpath "$0"))"

SRCDIR="$(realpath "$1")"
DESTDIR="$(realpath "$2")"

[ -n "$SRCDIR" ]
[ -d "$SRCDIR" ]

[ -n "$DESTDIR" ]

find "$SRCDIR" -name 'mail.log.*' | while read FILEPATH; do
  echo "Cleaning $FILEPATH"
  FILE="$DESTDIR/${FILEPATH##${SRCDIR}}"
  mkdir -p "$(dirname "$FILE")"
  "${UTILS_DIR}/log_cleaner.py" < "$FILEPATH" > "$FILE"
done
