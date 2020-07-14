#!/bin/sh

set -o pipefail
set -e
set -vx

VERSION="$(cat VERSION.txt)"

ARTIFACT="lightmeter-linux_amd64-$VERSION"

mv lightmeter "$ARTIFACT"

./jfrog bt u --publish --user="$BINTRAY_USER" --key="$BINTRAY_KEY" "$ARTIFACT"  "lightmeter/controlcenter/controlcenter/$VERSION"
