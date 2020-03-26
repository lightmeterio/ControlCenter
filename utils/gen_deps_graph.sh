#!/bin/bash

echo "digraph {"
go mod graph | sed 's|^\(.\+\) \(.\+\)$|  "\1" -> "\2";|g'
echo "}"
