#!/bin/bash
# Automated Web Channel Test Script
# Usage: ./test-automated.sh

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}PicoClaw Web Channel Automated Test${NC}"
echo -e "${GREEN}========================================${NC}\n"

# Test configuration
HOST="127.0.0.1"
PORT="8080"
BASE_URL="http://$HOST:$PORT"
AUTH_TOKEN="test-secret-token-123"
BINARY="../../build/picoclaw"

# Test counters
PASS=0
FAIL=0

# Helper functions
test_http() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4
    local expected_code=$5
    
    echo -ne "${YELLOW}Testing: $name...${NC} "
    
    if [ -n "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X $method \
            -H "Content-Type: application/json" \
            -d "$data" \
            -c cookies.txt -b cookies.txt \
            "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X $method \
            -c cookies.txt -b cookies.txt \
            "$BASE_URL$endpoint")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$http_code" -eq "$expected_code" ]; then
        echo -e "${GREEN}PASS${NC} (HTTP $http_code)"
        ((PASS++))
        return 0
    else
        echo -e "${RED}FAIL${NC} (Expected $expected_code, got $http_code)"
        echo "Response: $body"
        ((FAIL++))
        return 1
    fi
}

# Build if needed
if [ ! -f "$BINARY" ]; then
    echo -e "${YELLOW}Building binary...${NC}"
    cd ../.. && make build && cd test/web-channel
fi

# Setup test environment
export PICOCLAW_CHANNELS_WEB_ENABLED=true
export PICOCLAW_CHANNELS_WEB_HOST=127.0.0.1
export PICOCLAW_CHANNELS_WEB_PORT=8080
export PICOCLAW_CHANNELS_WEB_AUTH_TOKEN="test-secret-token-123"
export PICOCLAW_CHANNELS_WEB_SESSION_MAX_AGE=3600

# Setup workspace
mkdir -p workspace/memory

# Start server
echo -e "${YELLOW}Starting test server...${NC}"
$BINARY gateway > test-server.log 2>&1 &
SERVER_PID=$!

# Wait for server
sleep 3

if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}Server failed to start!${NC}"
    cat test-server.log
    exit 1
fi

echo -e "${GREEN}✓ Server started (PID: $SERVER_PID)${NC}\n"

# Cleanup function
cleanup() {
    echo -e "\n${YELLOW}Cleaning up...${NC}"
    kill $SERVER_PID 2>/dev/null || true
    rm -f cookies.txt test-server.log
}

trap cleanup EXIT

# Run tests
echo -e "${GREEN}Running HTTP tests...${NC}\n"

# Test 1: Index page
test_http "GET /" "GET" "/" "" 200

# Test 2: Status without auth (should work, no auth configured)
test_http "GET /api/status (no auth)" "GET" "/api/status" "" 200

# Test 3: Auth with wrong token
test_http "POST /auth (wrong token)" "POST" "/auth" '{"token":"wrong"}' 401

# Test 4: Auth with correct token
test_http "POST /auth (correct token)" "POST" "/auth" "{\"token\":\"$AUTH_TOKEN\"}" 200

# Test 5: Status with valid session
test_http "GET /api/status (with session)" "GET" "/api/status" "" 200

# Test 6: WebSocket upgrade (just check endpoint exists)
test_http "GET /ws (should upgrade)" "GET" "/ws" "" 400

echo -e "\n${GREEN}Running unit tests...${NC}"
cd ../..
go test -v -cover ./pkg/channels -run TestWeb

# Results
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}Test Results${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo -e "${GREEN}========================================${NC}\n"

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi
