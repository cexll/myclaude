#!/bin/bash
# Benchmark script for Claude CLI stability test
# Tests if the stdoutDrainTimeout fix resolves intermittent failures

set -euo pipefail

RUNS=${1:-100}
FAIL_COUNT=0
SUCCESS_COUNT=0
TIMEOUT_COUNT=0

echo "Running $RUNS iterations..."
echo "---"

for i in $(seq 1 $RUNS); do
    result=$(timeout 30 codeagent --backend claude --skip-permissions 'say OK' 2>&1) || true

    if echo "$result" | grep -q 'without agent_message'; then
        ((FAIL_COUNT++))
        echo "[$i] FAIL: without agent_message"
    elif echo "$result" | grep -q 'timeout'; then
        ((TIMEOUT_COUNT++))
        echo "[$i] TIMEOUT"
    elif echo "$result" | grep -q 'OK\|ok'; then
        ((SUCCESS_COUNT++))
        printf "\r[$i] OK                    "
    else
        ((FAIL_COUNT++))
        echo "[$i] FAIL: unexpected output"
        echo "$result" | head -3
    fi
done

echo ""
echo "---"
echo "Results ($RUNS runs):"
echo "  Success: $SUCCESS_COUNT ($(echo "scale=1; $SUCCESS_COUNT * 100 / $RUNS" | bc)%)"
echo "  Fail:    $FAIL_COUNT ($(echo "scale=1; $FAIL_COUNT * 100 / $RUNS" | bc)%)"
echo "  Timeout: $TIMEOUT_COUNT ($(echo "scale=1; $TIMEOUT_COUNT * 100 / $RUNS" | bc)%)"

if [ $FAIL_COUNT -gt 0 ]; then
    exit 1
fi
