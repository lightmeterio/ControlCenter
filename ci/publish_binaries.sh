#!/bin/sh

set -o pipefail
set -e
set -vx

VERSION="$(cat VERSION.txt)"

ARTIFACT="lightmeter-linux_amd64-$VERSION"

mv lightmeter "$ARTIFACT"

curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file "$ARTIFACT" \
  "${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/lightmeter/$VERSION/$ARTIFACT"
