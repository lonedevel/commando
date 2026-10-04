#!/bin/sh
# Prints the version declared in cmd/commando/main.go, e.g. 0.2.0.
set -eu
sed -n 's/^var version = "\(.*\)"$/\1/p' "$(dirname "$0")/../cmd/commando/main.go"
