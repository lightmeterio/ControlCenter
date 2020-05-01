#!/bin/sh

set -o pipefail
set -e
set -vx

mkdir -p .docker-cache

/kaniko/executor \
  --context $CI_PROJECT_DIR \
  --dockerfile $CI_PROJECT_DIR/ci/Dockerfile \
  --no-push \
  --cache-dir .docker-cache
