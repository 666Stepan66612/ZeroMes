# Group Chats Encryption Protocol

## Overview

Group chats in ZeroMes use a shared seed-based encryption scheme that maintains zero-knowledge architecture while providing scalability for large groups. Each group has a single cryptographic seed that is encrypted individually for each member using ECDH key exchange.

## Key Derivation

### Group Key Seed

When a group is created, a 32-byte random seed is generated:

```
group_key_seed = random(32 bytes)
```

This seed remains constant throughout the group's lifetime and is never changed.

### Message Encryption Keys

For each message, a unique encryption key is derived from the seed and current key version:

```
message_key = SHA256(group_key_seed || key_version)
```

Where `key_version` is an integer that increments when a member leaves or is removed from the group.

## Encryption Protocol

### Group Creation

1. Creator generates `group_key_seed`
2. For each initial member (including creator):
   - Compute `shared_secret = ECDH(creator_private, member_public)`
   - Encrypt seed: `encrypted_seed = AES-GCM(group_key_seed, shared_secret)`
   - Store in database: `user_group_keys(user_id, group_id, encrypted_seed, encrypted_by, key_version)`
3. Set initial `key_version = 0`

### Adding Member

1. Any authorized member can add a new user
2. Current member retrieves their decrypted seed from memory
3. Compute `shared_secret = ECDH(adder_private, new_member_public)`
4. Encrypt seed: `encrypted_seed = AES-GCM(group_key_seed, shared_secret)`
5. Store encrypted seed for new member
6. Set `can_read_from_message_id` to current last message ID
7. Key version remains unchanged

New member can only decrypt messages sent after joining.

### Sending Message

1. Retrieve encrypted seed from server
2. Decrypt: `seed = AES-GCM.decrypt(encrypted_seed, ECDH(my_private, encryptor_public))`
3. Get current `key_version` from server
4. Derive message key: `message_key = SHA256(seed || key_version)`
5. Encrypt message: `ciphertext = AES-GCM(plaintext, message_key)`
6. Send to server with `group_id` and `key_version`

### Receiving Message

1. Retrieve encrypted seed (if not in memory)
2. Decrypt seed using ECDH
3. Derive message key: `message_key = SHA256(seed || msg.key_version)`
4. Decrypt: `plaintext = AES-GCM.decrypt(msg.ciphertext, message_key)`

### Removing Member or Member Leaving

1. Member is marked as left in database: `UPDATE group_members SET left_at = NOW()`
2. Database trigger sets `needs_rotation = TRUE` on group
3. First online member acquires rotation lock
4. Increment `key_version++`
5. Update group record with new version
6. Release lock

All remaining members continue using the same seed but with incremented version. Removed member can still derive future keys by incrementing version themselves, but this is accepted as a reasonable security trade-off.

## Key Rotation

Key rotation occurs only when a member leaves or is removed. The rotation increments the version counter:

```
key_version(t+1) = key_version(t) + 1
```

This provides forward secrecy for messages sent before the member joined, and backward secrecy for messages sent after they left, under the assumption that the member does not retain the seed after leaving.

## Database Schema

### group_chats
- `id`: UUID, primary key
- `name`: text
- `key_version`: integer, current version
- `needs_rotation`: boolean, rotation flag
- `created_by`: UUID, creator user ID
- `created_at`: timestamp

### group_members
- `group_id`: UUID, foreign key
- `user_id`: UUID
- `role`: text, 'admin' or 'member'
- `can_read_from_message_id`: UUID, read permission boundary
- `joined_at`: timestamp
- `left_at`: timestamp, null if active

### user_group_keys
- `user_id`: UUID
- `group_id`: UUID
- `encrypted_seed`: text, AES-GCM encrypted
- `encrypted_by`: UUID, encryptor's user ID for ECDH
- `key_version`: integer

### messages
- `group_id`: UUID, null for 1-1 chats
- `key_version`: integer, encryption key version
- `recipient_id`: UUID, null for group messages

## Security Properties

### Zero-Knowledge

The server stores only encrypted seeds. Each seed is encrypted using ECDH with the recipient's public key, making it impossible for the server to decrypt without the recipient's private key.

### Forward Secrecy

New members cannot decrypt messages sent before they joined, as their `can_read_from_message_id` restriction is enforced server-side.

### Partial Backward Secrecy

Incrementing the key version after a member leaves provides backward secrecy against passive attacks. However, a malicious member who retains the seed can compute future keys by incrementing the version counter. This is accepted as:

1. A motivated attacker can simply remain in the group silently
2. Social engineering attacks (asking current members for keys) are always possible
3. The alternative (N-party Diffie-Hellman) is complex and fails if any member is offline

### Scalability

Message encryption and decryption are O(1) operations regardless of group size. Each member stores only their encrypted seed, not keys for all other members.

## Access Control

Group settings define permissions:
- `who_can_add_members`: 'admins' or 'anyone'
- `who_can_remove_members`: 'admins' (default)
- `who_can_send_messages`: 'anyone' (default)

These are enforced at the service layer before cryptographic operations.

## Implementation Notes

### Seed Storage

Decrypted seeds are stored in memory (RAM) for the duration of the user session. On logout or session end, all seeds are cleared. If "Remember Me" is enabled, seeds are not persisted; only the user's private key is stored (encrypted with password-derived key).

### Key Version Synchronization

Clients must fetch the current `key_version` before sending messages to ensure consistency. If a message arrives with a higher version than locally cached, the client updates its local version.

### Rotation Lock

Database-level locking prevents concurrent rotation attempts:

```sql
UPDATE group_chats
SET rotation_in_progress = TRUE, rotation_by = $user_id
WHERE group_id = $group_id AND rotation_in_progress = FALSE
```

Only one member can perform rotation at a time.

## Message Flow

### Group Message Send

1. Frontend: Derive message key from seed and version
2. Frontend: Encrypt message with AES-GCM
3. WebSocket: Send to api-gateway
4. API Gateway: Forward to message-service via gRPC
5. Message Service: Store in PostgreSQL, publish to Kafka
6. Realtime Service: Consume from Kafka, push to all online group members via WebSocket

### Group Message Receive

1. Realtime Service: Push message to client via WebSocket
2. Frontend: Check if seed is in memory
3. If not: Fetch encrypted seed from server, decrypt via ECDH
4. Frontend: Derive message key from seed and message's key_version
5. Frontend: Decrypt message with AES-GCM
6. Frontend: Display plaintext

## Comparison with Alternative Approaches

### Sender Keys (Signal Protocol)

Each member maintains a separate chain key. Messages are encrypted once but require storing N keys per client (one per member).

- Pros: True forward secrecy per-message
- Cons: O(N) memory per client, complex ratcheting logic

### Shared Group Key with Periodic Rotation

Single key rotated on a schedule (e.g., daily).

- Pros: Simple implementation
- Cons: Arbitrary rotation schedule, coordination overhead

### Per-Member Encryption

Each message encrypted N times, once per member.

- Pros: Maximum security, per-member forward secrecy
- Cons: O(N) encryption time, O(N) message size

### Chosen Approach: Seed-Based with Event-Driven Rotation

Balances security, scalability, and implementation complexity. Suitable for groups up to thousands of members.
