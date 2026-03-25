#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== WebSocket Test Script ===${NC}\n"

# Read saved data
ALICE_ID=$(cat /tmp/alice1-id.txt)
BOB_ID=$(cat /tmp/bob1-id.txt)
ALICE_TOKEN=$(cat /tmp/alice1-token.txt)
BOB_TOKEN=$(cat /tmp/bob1-token.txt)

echo -e "${GREEN}alice1 ID: ${ALICE_ID}${NC}"
echo -e "${GREEN}bob1 ID: ${BOB_ID}${NC}\n"

# Create named pipes for WebSocket communication
mkfifo /tmp/alice1-ws-in 2>/dev/null || true
mkfifo /tmp/bob1-ws-in 2>/dev/null || true

echo -e "${BLUE}Step 1: Connecting alice1 to WebSocket...${NC}"
# Start alice1 WebSocket connection in background
websocat -H="Cookie: access_token=${ALICE_TOKEN}" \
  "ws://localhost:8083/ws" \
  < /tmp/alice1-ws-in \
  > /tmp/alice1-ws-out.log 2>&1 &
ALICE_WS_PID=$!
sleep 1

echo -e "${BLUE}Step 2: Connecting bob1 to WebSocket...${NC}"
# Start bob1 WebSocket connection in background
websocat -H="Cookie: access_token=${BOB_TOKEN}" \
  "ws://localhost:8083/ws" \
  < /tmp/bob1-ws-in \
  > /tmp/bob1-ws-out.log 2>&1 &
BOB_WS_PID=$!
sleep 1

echo -e "${GREEN}WebSocket connections established${NC}\n"

# Test 1: alice1 sends message to bob1
echo -e "${BLUE}Step 3: alice1 sends message to bob1...${NC}"
echo "{\"type\":\"send_message\",\"recipient_id\":\"${BOB_ID}\",\"content\":\"Hello bob1!\",\"message_type\":\"text\"}" > /tmp/alice1-ws-in
sleep 2

# Check bob1 received the message
echo -e "${GREEN}bob1 received:${NC}"
tail -5 /tmp/bob1-ws-out.log

# Save message ID from alice's perspective
MESSAGE_ID=$(tail -20 /tmp/alice1-ws-out.log | grep -o '"message_id":"[^"]*"' | head -1 | cut -d'"' -f4)
echo -e "\n${GREEN}Message ID: ${MESSAGE_ID}${NC}\n"

# Test 2: bob1 marks message as read (THIS SHOULD BE DELIVERED TO BOTH)
echo -e "${BLUE}Step 4: bob1 marks message as read...${NC}"
CHAT_ID="${ALICE_ID}_${BOB_ID}"
echo "{\"type\":\"mark_as_read\",\"chat_id\":\"${CHAT_ID}\",\"message_id\":\"${MESSAGE_ID}\"}" > /tmp/bob1-ws-in
sleep 2

echo -e "${GREEN}alice1 should receive mark_as_read event:${NC}"
tail -10 /tmp/alice1-ws-out.log | grep -A5 "message_read" || echo "NOT RECEIVED!"

echo -e "\n${GREEN}bob1 also receives confirmation:${NC}"
tail -10 /tmp/bob1-ws-out.log | grep -A5 "message_read" || echo "NOT RECEIVED!"

# Test 3: alice1 alters her message (THIS SHOULD BE DELIVERED TO BOTH)
echo -e "\n${BLUE}Step 5: alice1 alters her message...${NC}"
echo "{\"type\":\"alter_message\",\"message_id\":\"${MESSAGE_ID}\",\"new_content\":\"Hello bob1! (edited)\"}" > /tmp/alice1-ws-in
sleep 2

echo -e "${GREEN}bob1 should receive message_altered event:${NC}"
tail -10 /tmp/bob1-ws-out.log | grep -A5 "message_altered" || echo "NOT RECEIVED!"

echo -e "\n${GREEN}alice1 also receives confirmation:${NC}"
tail -10 /tmp/alice1-ws-out.log | grep -A5 "message_altered" || echo "NOT RECEIVED!"

# Test 4: bob1 sends reply
echo -e "\n${BLUE}Step 6: bob1 sends reply to alice1...${NC}"
echo "{\"type\":\"send_message\",\"recipient_id\":\"${ALICE_ID}\",\"content\":\"Hi alice1!\",\"message_type\":\"text\"}" > /tmp/bob1-ws-in
sleep 2

BOB_MESSAGE_ID=$(tail -20 /tmp/bob1-ws-out.log | grep -o '"message_id":"[^"]*"' | tail -1 | cut -d'"' -f4)
echo -e "${GREEN}Bob's message ID: ${BOB_MESSAGE_ID}${NC}"

# Test 5: bob1 deletes his message (THIS SHOULD BE DELIVERED TO BOTH)
echo -e "\n${BLUE}Step 7: bob1 deletes his message...${NC}"
echo "{\"type\":\"delete_message\",\"message_id\":\"${BOB_MESSAGE_ID}\"}" > /tmp/bob1-ws-in
sleep 2

echo -e "${GREEN}alice1 should receive message_deleted event:${NC}"
tail -10 /tmp/alice1-ws-out.log | grep -A5 "message_deleted" || echo "NOT RECEIVED!"

echo -e "\n${GREEN}bob1 also receives confirmation:${NC}"
tail -10 /tmp/bob1-ws-out.log | grep -A5 "message_deleted" || echo "NOT RECEIVED!"

# Cleanup
echo -e "\n${BLUE}Cleaning up...${NC}"
kill $ALICE_WS_PID $BOB_WS_PID 2>/dev/null
rm -f /tmp/alice1-ws-in /tmp/bob1-ws-in

echo -e "\n${GREEN}=== Test Complete ===${NC}"
echo -e "${BLUE}Full logs saved to:${NC}"
echo "  /tmp/alice1-ws-out.log"
echo "  /tmp/bob1-ws-out.log"
