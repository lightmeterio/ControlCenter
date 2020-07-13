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

# part of image registry.gitlab.com/gitlab-org/release-cli
release-cli create \
  --name "ControlCenter $VERSION" \
  --description "$(cat $RELEASE_NOTES)" \
  --tag-name "release/$VERSION" \
  --ref $CI_COMMIT_SHA \
  --assets-links-name "Binary for linux amd64" \
  --assets-links-url "https://bintray.com/lightmeter/controlcenter/download_file?file_path=lightmeter-linux_amd64-$VERSION"
