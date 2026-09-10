#!/usr/bin/env sh
set -eu

plugin_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output_dir="${REDAN_FAQ_BIN_DIR:-$plugin_root/bin}"
mkdir -p "$output_dir"
go build -trimpath -o "$output_dir/redan" "$plugin_root/cmd/redan-faq"
printf '%s\n' "$output_dir/redan"
