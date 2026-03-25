#!/usr/bin/env python3
import asyncio
import websockets
import json
import sys

# Read saved data
with open('/tmp/alice1-id.txt') as f:
    ALICE_ID = f.read().strip()
with open('/tmp/bob1-id.txt') as f:
    BOB_ID = f.read().strip()
with open('/tmp/alice1-token.txt') as f:
    ALICE_TOKEN = f.read().strip()
with open('/tmp/bob1-token.txt') as f:
    BOB_TOKEN = f.read().strip()

print(f"alice1 ID: {ALICE_ID}")
print(f"bob1 ID: {BOB_ID}\n")

alice_messages = []
bob_messages = []
message_id = None
bob_message_id = None

async def alice_client():
    global message_id
    uri = "ws://localhost:8083/ws"
    headers = {"Cookie": f"access_token={ALICE_TOKEN}"}
    
    async with websockets.connect(uri, extra_headers=headers) as ws:
        print("✓ alice1 connected to WebSocket")
        
        # Receive connection confirmation
        msg = await ws.recv()
        alice_messages.append(json.loads(msg))
        print(f"alice1 <- {msg}")
        
        # Wait a bit for bob to connect
        await asyncio.sleep(1)
        
        # Send message to bob
        print(f"\n→ alice1 sends message to bob1")
        send_msg = {
            "type": "send_message",
            "recipient_id": BOB_ID,
            "content": "Hello bob1!",
            "message_type": "text"
        }
        await ws.send(json.dumps(send_msg))
        
        # Receive response
        msg = await ws.recv()
        data = json.loads(msg)
        alice_messages.append(data)
        print(f"alice1 <- {msg}")
        
        if data.get("type") == "message_sent":
            message_id = data["payload"]["id"]
            print(f"✓ Message ID: {message_id}")
        
        # Wait for mark_as_read event from bob
        await asyncio.sleep(2)
        
        # Alter message
        print(f"\n→ alice1 alters message")
        alter_msg = {
            "type": "alter_message",
            "message_id": message_id,
            "new_content": "Hello bob1! (edited)"
        }
        await ws.send(json.dumps(alter_msg))
        
        # Keep receiving messages for a while
        try:
            for _ in range(10):
                msg = await asyncio.wait_for(ws.recv(), timeout=1.0)
                data = json.loads(msg)
                alice_messages.append(data)
                print(f"alice1 <- {msg}")
        except asyncio.TimeoutError:
            pass
        
        print("alice1 disconnecting...")

async def bob_client():
    global bob_message_id
    uri = "ws://localhost:8083/ws"
    headers = {"Cookie": f"access_token={BOB_TOKEN}"}
    
    await asyncio.sleep(0.5)  # Let alice connect first
    
    async with websockets.connect(uri, extra_headers=headers) as ws:
        print("✓ bob1 connected to WebSocket")
        
        # Receive connection confirmation
        msg = await ws.recv()
        bob_messages.append(json.loads(msg))
        print(f"bob1 <- {msg}")
        
        # Wait for alice's message
        msg = await ws.recv()
        data = json.loads(msg)
        bob_messages.append(data)
        print(f"\nbob1 <- {msg}")
        
        if data.get("type") == "new_message":
            msg_id = data["payload"]["message_id"]
            chat_id = data["payload"]["chat_id"]
            
            # Mark as read
            print(f"\n→ bob1 marks message as read")
            mark_read = {
                "type": "mark_as_read",
                "chat_id": chat_id,
                "last_message_id": msg_id
            }
            await ws.send(json.dumps(mark_read))
            
            # Receive confirmation
            msg = await ws.recv()
            data = json.loads(msg)
            bob_messages.append(data)
            print(f"bob1 <- {msg}")
        
        # Send reply
        await asyncio.sleep(1)
        print(f"\n→ bob1 sends reply to alice1")
        reply_msg = {
            "type": "send_message",
            "recipient_id": ALICE_ID,
            "content": "Hi alice1!",
            "message_type": "text"
        }
        await ws.send(json.dumps(reply_msg))
        
        msg = await ws.recv()
        data = json.loads(msg)
        bob_messages.append(data)
        print(f"bob1 <- {msg}")
        
        if data.get("type") == "message_sent":
            bob_message_id = data["payload"]["id"]
        
        # Wait for alice's alter event
        await asyncio.sleep(2)
        
        # Delete own message
        print(f"\n→ bob1 deletes his message")
        delete_msg = {
            "type": "delete_message",
            "message_id": bob_message_id
        }
        await ws.send(json.dumps(delete_msg))
        
        # Keep receiving messages
        try:
            for _ in range(10):
                msg = await asyncio.wait_for(ws.recv(), timeout=1.0)
                data = json.loads(msg)
                bob_messages.append(data)
                print(f"bob1 <- {msg}")
        except asyncio.TimeoutError:
            pass
        
        print("bob1 disconnecting...")

async def main():
    print("=== Starting WebSocket Test ===\n")
    
    # Run both clients concurrently
    await asyncio.gather(
        alice_client(),
        bob_client()
    )
    
    print("\n=== Test Results ===\n")
    
    # Check if alice received mark_as_read event
    alice_got_read = any(m.get("type") == "message_read" for m in alice_messages)
    print(f"✓ alice1 received mark_as_read event: {alice_got_read}")
    
    # Check if bob received mark_as_read event
    bob_got_read = any(m.get("type") == "message_read" for m in bob_messages)
    print(f"✓ bob1 received mark_as_read event: {bob_got_read}")
    
    # Check if bob received message_altered event
    bob_got_altered = any(m.get("type") == "message_altered" for m in bob_messages)
    print(f"✓ bob1 received message_altered event: {bob_got_altered}")
    
    # Check if alice received message_altered event
    alice_got_altered = any(m.get("type") == "message_altered" for m in alice_messages)
    print(f"✓ alice1 received message_altered event: {alice_got_altered}")
    
    # Check if alice received message_deleted event
    alice_got_deleted = any(m.get("type") == "message_deleted" for m in alice_messages)
    print(f"✓ alice1 received message_deleted event: {alice_got_deleted}")
    
    # Check if bob received message_deleted event
    bob_got_deleted = any(m.get("type") == "message_deleted" for m in bob_messages)
    print(f"✓ bob1 received message_deleted event: {bob_got_deleted}")
    
    print("\n=== Dual Delivery Test Results ===")
    print(f"mark_as_read delivered to BOTH: {alice_got_read and bob_got_read}")
    print(f"message_altered delivered to BOTH: {alice_got_altered and bob_got_altered}")
    print(f"message_deleted delivered to BOTH: {alice_got_deleted and bob_got_deleted}")

if __name__ == "__main__":
    asyncio.run(main())
