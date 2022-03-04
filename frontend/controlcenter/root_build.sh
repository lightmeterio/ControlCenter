#!/bin/sh

set -e

cd ./frontend/controlcenter
vue build --dest ../../www ./src/main.js
