#!/bin/bash
# Smoke test for the agentpack CLI. Run from repo root: bash tests/smoke.sh
set -u
export GOTOOLCHAIN=${GOTOOLCHAIN:-go1.24.6}
BIN=$(mktemp -d)/agentpack
go build -o "$BIN" ./cmd/agentpack || exit 1
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT

"$BIN" init "$T/a" --name demo || exit 1
"$BIN" run --dir "$T/a" --dry-run > /dev/null || exit 1
"$BIN" diff examples/research-assistant/agentpack.yaml "$T/a/agentpack.yaml" > /dev/null || exit 1
"$BIN" --json diff examples/research-assistant/agentpack.yaml "$T/a/agentpack.yaml" > /dev/null || exit 1
# invalid manifest must fail with structured error
echo "name: Bad!" > "$T/bad.yaml"
if "$BIN" --json diff examples/research-assistant/agentpack.yaml "$T/bad.yaml" 2>/dev/null; then
  echo "expected diff to fail on invalid manifest"; exit 1
fi
"$BIN" init "$T/oc" --name oc-agent --harness opencode || exit 1
"$BIN" run --dir "$T/oc" --dry-run > /dev/null || exit 1
if "$BIN" init "$T/nope" --harness bogus 2>/dev/null; then
  echo "expected init to reject bogus harness"; exit 1
fi
echo "smoke OK"
