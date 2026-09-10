#!/usr/bin/env sh
set -eu

plugin_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output_dir="$plugin_root/bin"
mkdir -p "$output_dir"
go build -trimpath -o "$output_dir/redan" "$plugin_root/cmd/redan"
printf '%s\n' "$output_dir/redan"
