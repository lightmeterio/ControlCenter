#!/bin/sh

set -o pipefail
set -e
set -vx

function image_tag_from_git_tag() {
  if [ -n "${CI_COMMIT_TAG}" ]; then
    echo "${CI_COMMIT_TAG#release/}"
  else
    echo "ERROR: we publish docker images only on new tags for now" >&2
    return 1
  fi
}

IMAGE_TAG=${SCHEDULED_DOCKER_IMAGE_TAG:-$(image_tag_from_git_tag)}

cat > /kaniko/.docker/config.json << EOF
{
  "auths": {
    "$CI_REGISTRY":{
      "username":"$CI_REGISTRY_USER",
      "password":"$CI_REGISTRY_PASSWORD"
    },

    "https://index.docker.io/v1/":{
      "username":"$DOCKER_IO_REGISTRY_USER",
      "password":"$DOCKER_IO_REGISTRY_PASSWORD"
    }
  }
}"
EOF

mkdir -p .docker-cache

# only define the latest tag for release images
if [ -z "${SCHEDULED_DOCKER_IMAGE_TAG}" ]; then
  EXTRA_DESTINATIONS="--destination $CI_REGISTRY_IMAGE:latest --destination index.docker.io/lightmeter/controlcenter:latest"
fi

/kaniko/executor \
  --context $CI_PROJECT_DIR \
  --dockerfile $CI_PROJECT_DIR/ci/Dockerfile \
  --destination $CI_REGISTRY_IMAGE:$IMAGE_TAG \
  --destination index.docker.io/lightmeter/controlcenter:$IMAGE_TAG \
  $EXTRA_DESTINATIONS \
  --build-arg "LIGHTMETER_VERSION=$(cat VERSION.txt)" \
  --build-arg "LIGHTMETER_COMMIT=$CI_COMMIT_SHA" \
  --build-arg "IMAGE_TAG=$IMAGE_TAG" \
  --cache-dir .docker-cache
