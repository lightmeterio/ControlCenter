#!/bin/bash

# Script to remove identifying information from standard mail log files generated by Postfix
# 
# Note: Requires the additional script `log_cleaner.py` to be located within the same path
# Usage: batch_log_cleaner /path/to/source/logs /path/to/destination/logs
#        e.g.  batch_log_cleaner /var/mail/log /home/admin/cleaned_logs

set -e
set -o pipefail

# Check user arguments are correct
if [ -z "$1" ]
  then
    echo "No mail log source directory path supplied"
    SHOWUSAGE=1
  fi
if [ -z "$2" ]
  then
    echo "No destination directory path supplied for putting cleaned files"
    SHOWUSAGE=1
fi

if [ $SHOWUSAGE == 1 ]
  then
    echo "Example usage: batch_log_cleaner /var/mail/log /home/admin/cleaned_logs"
    exit 1
fi

# Alias argument vars
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
