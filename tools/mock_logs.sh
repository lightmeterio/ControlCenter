#!/usr/bin/env bash

# This script generate the right time for postfix log lines
# based in the current time.
# It's indended to be run during manual tests, where some log lines can be input
# but its time must be the current one in order for the application to run properly

# Pipe this script to controlcenter stdin, like in:
# tools/mock_logs.sh | ./lightmeter -workspace some_workspace -stdin
# And then paste logs without the first part of the line, with the time.
#
# For instance, for the original log line "Jan 10 17:24:24 mail postfix/smtp[8942]: 704042C00397: some content",
# you should paste "mail postfix/smtp[8942]: 704042C00397: some content" that this script
# will fill the beginning of the line with the correct time.

# Funny how the documentation is longer than the implementation, right? :-)

while read line; do
  echo "$(date '+%b %_2d %H:%M:%S') $line"
done
