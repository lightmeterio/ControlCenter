#!/bin/sh
echo "po/en/LC_MESSAGES -> .pot"
go run tools/code2po/main.go -i www -pot -o po/controlcenter.pot
for d in po/*; do
  if [ -d "$d/LC_MESSAGES" ] && [ $d != "po/en" ]; then
    echo "$d/LC_MESSAGES -> .po"
    go run tools/code2po/main.go -i www -o "$d/LC_MESSAGES/controlcenter.po" -ids=true
  fi
done
echo "po/en/LC_MESSAGES -> .po"
go run tools/code2po/main.go -i www -o po/en/LC_MESSAGES/controlcenter.po
