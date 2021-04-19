#!/bin/sh

set -o pipefail
set -e
set -vx

VERSION=$(cat VERSION.txt)
RELEASE_NOTES="release_notes/$VERSION"

if [ ! -e "$RELEASE_NOTES" ]; then
  echo "ERROR: Could not find a file $RELEASE_NOTES"
  exit 1
fi

ARTIFACT="lightmeter-linux_amd64-$VERSION"
CHECKSUM=sha256.txt

apk add curl

cp lightmeter "$ARTIFACT"

sha256sum "$ARTIFACT" >> "$CHECKSUM"

ARTIFACT_URL="${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/lightmeter/$VERSION/$ARTIFACT"
CHECKSUM_URL="${CI_API_V4_URL}/projects/${CI_PROJECT_ID}/packages/generic/lightmeter/$VERSION/$CHECKSUM"

# first, upload stuff to the package registry
curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file "$ARTIFACT" "$ARTIFACT_URL"
curl --header "JOB-TOKEN: $CI_JOB_TOKEN" --upload-file "$CHECKSUM" "$CHECKSUM_URL"

# and finally perform the release

release-cli create \
  --name "ControlCenter $VERSION" \
  --description "$(cat $RELEASE_NOTES)" \
  --tag-name "release/$VERSION" \
  --ref $CI_COMMIT_SHA \
  --assets-links-name "Binary for Linux amd64" \
  --assets-links-url "$ARTIFACT_URL" \
  --assets-links-name "Binary for Linux amd64 (sha256sum)" \
  --assets-links-url "$CHECKSUM_URL"
