#!/bin/sh

set -e

cd ./frontend/controlcenter

echo "Version: " $1

export VUE_APP_VERSION=$1 && vue build --dest ../../www ./src/main.js
