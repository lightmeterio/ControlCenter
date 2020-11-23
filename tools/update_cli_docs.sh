#!/bin/sh

# TODO: Generate a format more suitable for being displayed online

set -e

make release

(
  echo '```'
  ./lightmeter -help 2>&1
  echo '```'
) > cli_usage.md
