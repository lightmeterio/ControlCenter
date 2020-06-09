#!/usr/bin/env python3

# Generate Graphviz graph from the output of `go mod graph`
# grouping by package, in case more than one version of the
# same dependency is used.
# Receives input via stdin
# Usage:
# go mod graph | tools/gen_deps_graph.py | dot -Tsvg > output.svg

import fileinput

groups = {}

pairs = []

def group(token):
    s = token.split("@")

    if s[0] not in groups:
        groups[s[0]] = set()

    groups[s[0]].add(token)

    return token

for line in fileinput.input():
    tokens = line.rstrip().split(" ")
    source = tokens[0]
    dest = tokens[1]
    s = group(source)
    d = group(dest)
    pairs += [(s, d)]

print("digraph {")

count = 0

for (k, v) in groups.items():
    if len(v) == 1:
        continue

    print(f'  subgraph cluster_{count} {{\n    label="{k}"')

    count += 1

    for i in v:
        l = i.split("@")[1]
        print(f'    "{i}" [label="{l}"]')

    print("  }")

for (s, d) in pairs:
    print(f'"{s}" -> "{d}"')

print("}")
