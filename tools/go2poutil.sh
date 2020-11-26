#!/bin/sh
echo "po/en/LC_MESSAGES -> .pot"
go run tools/go2po/main.go -i . -o po/backend.pot
for d in po/*; do
  if [ -d "$d/LC_MESSAGES" ] && [ $d != "po/en" ]; then
    echo "$d/LC_MESSAGES -> .po"
    go run tools/go2po/main.go -i . -o "$d/LC_MESSAGES/backend.po" -ids=true
  fi
done
echo "po/en/LC_MESSAGES -> .po"
go run tools/go2po/main.go -i . -o po/en/LC_MESSAGES/backend.po