#!/bin/bash
# Integration tests for mix-cli

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
BUILD_DIR="$ROOT_DIR/build"
MIX_CLI="$BUILD_DIR/mix-cli/mix-cli"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

log_pass() { 
    echo -e "${GREEN}[PASS]${NC} $1"
    ((TESTS_PASSED++))
}

log_fail() { 
    echo -e "${RED}[FAIL]${NC} $1"
    ((TESTS_FAILED++))
}

log_skip() { 
    echo -e "${YELLOW}[SKIP]${NC} $1"
}

echo "=========================================="
echo "mix-cli Integration Tests"
echo "=========================================="

# Check if mix-cli exists
if [ ! -f "$MIX_CLI" ]; then
    echo "mix-cli not found at $MIX_CLI"
    echo "Building mix-cli..."
    cd "$ROOT_DIR/mix-cli"
    go build -o "$MIX_CLI" . 2>/dev/null || {
        log_fail "Failed to build mix-cli"
        exit 1
    }
fi

# Test: Version
echo ""
echo "Test: mix --version"
if $MIX_CLI --version 2>&1 | grep -q "Version"; then
    log_pass "Version command works"
else
    log_fail "Version command failed"
fi

# Test: Help
echo ""
echo "Test: mix --help"
if $MIX_CLI --help 2>&1 | grep -q "MIXOS"; then
    log_pass "Help command works"
else
    log_fail "Help command failed"
fi

# Test: Agent subcommand exists
echo ""
echo "Test: mix agent --help"
if $MIX_CLI agent --help 2>&1 | grep -q "agent"; then
    log_pass "Agent subcommand exists"
else
    log_fail "Agent subcommand missing"
fi

# Test: VM subcommand exists
echo ""
echo "Test: mix vm --help"
if $MIX_CLI vm --help 2>&1 | grep -q "vm\|VM"; then
    log_pass "VM subcommand exists"
else
    log_fail "VM subcommand missing"
fi

# Test: Container subcommand exists
echo ""
echo "Test: mix container --help"
if $MIX_CLI container --help 2>&1 | grep -q "container"; then
    log_pass "Container subcommand exists"
else
    log_fail "Container subcommand missing"
fi

# Test: Dev subcommand exists
echo ""
echo "Test: mix dev --help"
if $MIX_CLI dev --help 2>&1 | grep -q "dev"; then
    log_pass "Dev subcommand exists"
else
    log_fail "Dev subcommand missing"
fi

# Test: Status command (may fail without system)
echo ""
echo "Test: mix status"
if $MIX_CLI status 2>&1; then
    log_pass "Status command runs"
else
    log_skip "Status command (requires system)"
fi

# Test: Agent status (may fail without agent running)
echo ""
echo "Test: mix agent status"
OUTPUT=$($MIX_CLI agent status 2>&1) || true
if echo "$OUTPUT" | grep -q "status\|running\|stopped\|error\|connect"; then
    log_pass "Agent status command runs"
else
    log_skip "Agent status (requires agent)"
fi

# Summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
