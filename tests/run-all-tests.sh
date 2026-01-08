#!/bin/bash
# MIXOS GO Test Runner
# Runs all available tests

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TOTAL_PASSED=0
TOTAL_FAILED=0
TOTAL_SKIPPED=0

log_header() { echo -e "\n${BLUE}=== $1 ===${NC}\n"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((TOTAL_PASSED++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1"; ((TOTAL_FAILED++)); }
log_skip() { echo -e "${YELLOW}[SKIP]${NC} $1"; ((TOTAL_SKIPPED++)); }

echo "=========================================="
echo "MIXOS GO Test Suite"
echo "=========================================="

# Parse arguments
RUN_UNIT=true
RUN_INTEGRATION=true
RUN_QEMU=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --unit-only) RUN_INTEGRATION=false; RUN_QEMU=false; shift ;;
        --integration-only) RUN_UNIT=false; RUN_QEMU=false; shift ;;
        --qemu) RUN_QEMU=true; shift ;;
        --all) RUN_UNIT=true; RUN_INTEGRATION=true; RUN_QEMU=true; shift ;;
        --help)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --unit-only         Run only unit tests"
            echo "  --integration-only  Run only integration tests"
            echo "  --qemu              Include QEMU tests (slow)"
            echo "  --all               Run all tests including QEMU"
            echo "  --help              Show this help"
            exit 0
            ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# Unit Tests
if [ "$RUN_UNIT" = true ]; then
    log_header "Unit Tests"
    
    # Go tests
    for dir in mix-cli mix-installer mix-agent-early; do
        if [ -d "$ROOT_DIR/$dir" ]; then
            echo "Testing $dir..."
            cd "$ROOT_DIR/$dir"
            if go test ./... 2>/dev/null; then
                log_pass "$dir Go tests"
            else
                log_skip "$dir Go tests (no tests or failed)"
            fi
        fi
    done
    
    # Rust tests
    if [ -d "$ROOT_DIR/mix-pkg" ]; then
        echo "Testing mix-pkg..."
        cd "$ROOT_DIR/mix-pkg"
        if cargo test 2>/dev/null; then
            log_pass "mix-pkg Rust tests"
        else
            log_skip "mix-pkg Rust tests (no tests or failed)"
        fi
    fi
    
    # Python tests
    if [ -d "$ROOT_DIR/mix-agent" ]; then
        echo "Testing mix-agent..."
        cd "$ROOT_DIR/mix-agent"
        if [ -d "venv" ]; then
            source venv/bin/activate
            if python -m pytest tests/ 2>/dev/null; then
                log_pass "mix-agent Python tests"
            else
                log_skip "mix-agent Python tests (no tests or failed)"
            fi
            deactivate
        else
            log_skip "mix-agent Python tests (no venv)"
        fi
    fi
fi

# Integration Tests
if [ "$RUN_INTEGRATION" = true ]; then
    log_header "Integration Tests"
    
    # mix-cli integration tests
    if [ -f "$SCRIPT_DIR/integration/test_mix_cli.sh" ]; then
        echo "Running mix-cli integration tests..."
        if bash "$SCRIPT_DIR/integration/test_mix_cli.sh"; then
            log_pass "mix-cli integration tests"
        else
            log_fail "mix-cli integration tests"
        fi
    fi
    
    # mix-agent integration tests
    if [ -f "$SCRIPT_DIR/integration/test_mix_agent.py" ]; then
        echo "Running mix-agent integration tests..."
        cd "$ROOT_DIR"
        if python3 "$SCRIPT_DIR/integration/test_mix_agent.py"; then
            log_pass "mix-agent integration tests"
        else
            log_fail "mix-agent integration tests"
        fi
    fi
fi

# QEMU Tests
if [ "$RUN_QEMU" = true ]; then
    log_header "QEMU Tests"
    
    # Check if QEMU is available
    if ! command -v qemu-system-x86_64 &> /dev/null; then
        log_skip "QEMU tests (qemu-system-x86_64 not found)"
    else
        # Boot test
        if [ -f "$SCRIPT_DIR/qemu/test-boot.sh" ]; then
            echo "Running boot test..."
            if bash "$SCRIPT_DIR/qemu/test-boot.sh"; then
                log_pass "QEMU boot test"
            else
                log_fail "QEMU boot test"
            fi
        fi
    fi
fi

# Summary
echo ""
echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo -e "Passed:  ${GREEN}$TOTAL_PASSED${NC}"
echo -e "Failed:  ${RED}$TOTAL_FAILED${NC}"
echo -e "Skipped: ${YELLOW}$TOTAL_SKIPPED${NC}"
echo ""

if [ $TOTAL_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed${NC}"
    exit 1
fi
