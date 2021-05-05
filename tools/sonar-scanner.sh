#!/bin/sh

set -e
set -vx

export

sonar-scanner -Dsonar.projectVersion=$(cat VERSION.txt) \
  -Dsonar.branch.target="${CI_MERGE_REQUEST_TARGET_BRANCH_NAME:-master}" \
  -Dsonar.branch.name="${CI_MERGE_REQUEST_SOURCE_BRANCH_NAME:-${CI_COMMIT_BRANCH}}"

