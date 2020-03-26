#!/bin/bash

set -e

export GO111MODULE=off
go-fuzz-build -libfuzzer
clang -fsanitize=fuzzer parser-fuzz.a -o fuzzer_clang
./fuzzer_clang corpus
