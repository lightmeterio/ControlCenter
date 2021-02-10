#!/bin/bash

# jsonxf is a very fast json formatter written in rust:
# https://github.com/gamache/jsonxf

while read line; do
  jsonxf <<< "$line"
done
