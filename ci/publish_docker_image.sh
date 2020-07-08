#!/bin/sh

set -o pipefail
set -e
set -vx

function image_tag() {
  if [ -n "${CI_COMMIT_TAG}" ]; then
    echo "${CI_COMMIT_TAG#release/}"
  else
    echo "ERROR: we publish docker images only on new tags for now" >&2
    return 1
  fi
}

IMAGE_TAG=$(image_tag)

cat > /kaniko/.docker/config.json << EOF
{
  "auths": {
    "$CI_REGISTRY":{
      "username":"$CI_REGISTRY_USER",
      "password":"$CI_REGISTRY_PASSWORD"
    }
  }
}"
EOF

mkdir -p .docker-cache

/kaniko/executor \
  --context $CI_PROJECT_DIR \
  --dockerfile $CI_PROJECT_DIR/ci/Dockerfile \
  --destination $CI_REGISTRY_IMAGE:$IMAGE_TAG \
  --destination $CI_REGISTRY_IMAGE:latest \
  --cache-dir .docker-cache
