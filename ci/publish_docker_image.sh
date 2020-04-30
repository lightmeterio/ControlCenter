#!/bin/sh

set -o pipefail
set -e

IMAGE_TAG=${CI_COMMIT_TAG:-bad-dev}

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

/kaniko/executor \
  --context $CI_PROJECT_DIR \
  --dockerfile $CI_PROJECT_DIR/ci/Dockerfile \
  --destination $CI_REGISTRY_IMAGE:$IMAGE_TAG \
  --cache-dir .docker-cache
