#!/bin/bash
# Web Channel Manual Test Script
# Usage: ./test-manual.sh

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}PicoClaw Web Channel Manual Test${NC}"
echo -e "${GREEN}========================================${NC}\n"

# Check if binary exists
BINARY="../../build/picoclaw"
if [ ! -f "$BINARY" ]; then
    echo -e "${YELLOW}Binary not found. Building...${NC}"
    cd ../.. && make build && cd test/web-channel
fi

# Create test workspace
echo -e "${YELLOW}Setting up test workspace...${NC}"
mkdir -p workspace/memory
echo "# Test Workspace" > workspace/README.md

# Start server
echo -e "${YELLOW}Starting PicoClaw server on http://127.0.0.1:8080${NC}"
echo -e "${YELLOW}Auth token: test-secret-token-123${NC}\n"

# Export test config via environment variables
export PICOCLAW_CHANNELS_WEB_ENABLED=true
export PICOCLAW_CHANNELS_WEB_HOST=127.0.0.1
export PICOCLAW_CHANNELS_WEB_PORT=8080
export PICOCLAW_CHANNELS_WEB_AUTH_TOKEN="test-secret-token-123"
export PICOCLAW_CHANNELS_WEB_SESSION_MAX_AGE=3600

# Export API key if needed
if [ -z "$ANTHROPIC_API_KEY" ]; then
    echo -e "${RED}Warning: ANTHROPIC_API_KEY not set${NC}"
    echo -e "${YELLOW}Set it with: export ANTHROPIC_API_KEY=your-key-here${NC}\n"
fi

# Start server
echo -e "${YELLOW}Running: $BINARY gateway${NC}"
$BINARY gateway &
SERVER_PID=$!

echo -e "${GREEN}Server PID: $SERVER_PID${NC}"

# Wait for server to start
sleep 2

# Check if server is running
if ! ps -p $SERVER_PID > /dev/null; then
    echo -e "${RED}Server failed to start!${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Server started successfully${NC}\n"

# Open browser
echo -e "${YELLOW}Opening test client in browser...${NC}"
if command -v open &> /dev/null; then
    open client.html
elif command -v xdg-open &> /dev/null; then
    xdg-open client.html
else
    echo -e "${YELLOW}Please open client.html in your browser${NC}"
fi

echo -e "\n${GREEN}Test Instructions:${NC}"
echo "1. Enter auth token: test-secret-token-123"
echo "2. Click 'Authenticate'"
echo "3. Try sending messages"
echo "4. Check WebSocket connection status"
echo "5. Test rate limiting (send >10 messages quickly)"
echo "6. Test session expiry (wait >1 hour)"
echo -e "\n${YELLOW}Press Ctrl+C to stop server${NC}\n"

# Trap Ctrl+C to cleanup
trap "echo -e '\n${YELLOW}Stopping server...${NC}'; kill $SERVER_PID 2>/dev/null; exit 0" INT

# Show server logs
tail -f ../../picoclaw.log 2>/dev/null || wait $SERVER_PID
