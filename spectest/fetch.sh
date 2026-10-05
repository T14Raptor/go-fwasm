#!/bin/sh
# Fetches the WebAssembly spec tests at a tag and converts them with wabt's
# wast2json into testdata/<tag>/. Needs git and wast2json.
#
#   ./fetch.sh            # wg-2.0
#
# wg-2.0 includes every MVP test. wg-1.0 itself uses assertion syntax that
# current wabt no longer reads.
set -eu

tag=${1:-wg-2.0}
here=$(cd "$(dirname "$0")" && pwd)
out="$here/testdata/$tag"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

git clone -q --depth 1 --branch "$tag" --filter=blob:none --sparse \
	https://github.com/WebAssembly/spec "$tmp/spec"
git -C "$tmp/spec" sparse-checkout set test/core

rm -rf "$out"
mkdir -p "$out"
for f in "$tmp"/spec/test/core/*.wast; do
	(cd "$out" && wast2json "$f" -o "$(basename "$f" .wast).json")
done
echo "$tag: $(ls "$out"/*.json | wc -l | tr -d ' ') scripts in $out"
