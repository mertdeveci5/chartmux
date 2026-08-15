#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
frontend_dir=$(dirname "$script_dir")
repo_dir=$(dirname "$frontend_dir")
output_dir="$frontend_dir/public/demos"
build_dir=$(mktemp -d "${TMPDIR:-/tmp}/chartmux-demos.XXXXXX")

cleanup() {
  rm -rf "$build_dir"
}
trap cleanup EXIT INT TERM

mkdir -p "$output_dir"
(
  cd "$repo_dir"
  go build -o "$build_dir/chartmux" ./cmd/chartmux
)

"$build_dir/chartmux" demo --list | while IFS= read -r demo; do
  env -u NO_COLOR CLICOLOR_FORCE=1 TERM=xterm-ghostty COLORTERM=truecolor \
    "$build_dir/chartmux" demo "$demo" \
    --export terminal \
    --width 80 \
    --height 14 | \
    sed 's/[[:blank:]]*$//' \
    > "$output_dir/$demo.ansi.txt"
done
