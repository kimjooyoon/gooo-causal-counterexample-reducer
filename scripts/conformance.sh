#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "$0")/.." && pwd)
output_dir=${1:-"$(mktemp -d)"}

mkdir -p "$output_dir"
cd "$repo_root"
go run ./cmd/gooo-causal-counterexample-reducer \
  -mode conformance \
  -contract .gooo/causal-counterexample-reducer.gooo \
  -fixtures . \
  -output "$output_dir"

test -s "$output_dir/conformance-report.json"
test -s "$output_dir/metrics.json"
test -s "$output_dir/proof-vectors.json"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  {
    echo '### causal counterexample reducer'
    echo 'The generator completed the fixed denominator and malformed-input checks.'
    echo "Artifacts: $output_dir"
  } >> "$GITHUB_STEP_SUMMARY"
fi
