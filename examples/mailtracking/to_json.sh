#!/bin/bash

while read line; do
  json_pp <<< "$line"
done
