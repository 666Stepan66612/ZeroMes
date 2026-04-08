# Code Review: Zero-Knowledge Messenger

## Executive Summary

This is a well-architected zero-knowledge end-to-end encrypted messenger with microservices architecture. The cryptographic implementation is solid, but there are several security concerns and areas for improvement.

**Overall Rating: 7/10**

---

## 🔐 Cryptography & Security

### ✅ Strengths

1. **Proper E2EE Implementation**
   - Uses secp256k1 (ECDH) for key exchange
   - AES-256-GCM for message encryption
   - Server never sees plaintext messages

2. **Key Derivation**
   - [`derivePrivateKey()`](frontend/src/lib/crypto/ecc.ts:17) uses 1M SHA-256 iterations
   - Good protection against brute-force attacks

3. **Authentication Security**
   - Client-side password hashing before transmission
   - Server-side bcrypt with additional salt
   - Double hashing prevents rainbow table attacks

### ⚠️ Critical Security Issues

#### 1. **Weak Key Derivation Function (HIGH PRIORITY)**
**Location:** [`frontend/src/lib/crypto/ecc.ts:17-26`](frontend/src/lib/crypto/ecc.ts:17)

```typescript
export function derivePrivateKey(password: string, iterations = 1000000): Uint8Array {
  let hash: Uint8Array = new TextEncoder().encode(password)
  
  for (let i = 0; i < iterations; i++) {
    hash = new Uint8Array(sha256(hash))
  }
  
  return hash
}
```

**Problem:** No salt used in key derivation. Same password = same private key for all users.

**Impact:** 
- Attackers can precompute keys for common passwords
- Users with same password have same private key
- Vulnerable to rainbow table attacks

**Fix:** Use PBKDF2 or Argon2 with unique salt per user:
```typescript
export async function derivePrivateKey(
  password: string, 
  salt: Uint8Array,
  iterations = 1000000
): Promise<Uint8Array> {
  const passwordBytes = new TextEncoder().encode(password);
  const keyMaterial = await crypto.subtle.importKey(
    'raw', passwordBytes, 'PBKDF2', false, ['deriveBits']
  );
  
  const bits = await crypto.subtle.deriveBits(
    { name: 'PBKDF2', salt, iterations, hash: 'SHA-256' },
    keyMaterial,
    256
  );
  
  return new Uint8Array(bits);
}
```

#### 2. **Insecure Storage Implementation (MEDIUM)**
**Location:** [`frontend/src/lib/crypto/secureStorage.ts:37-70`](frontend/src/lib/crypto/secureStorage.ts:37)

```typescript
async function deriveStorageKey(): Promise<CryptoKey> {
  const fingerprint = `${navigator.userAgent}|${navigator.language}|${screen.width}x${screen.height}`;
  // ...
  const salt = new Uint8Array([
    0x73, 0x65, 0x63, 0x75, 0x72, 0x65, 0x6d, 0x65,
    0x73, 0x73, 0x65, 0x6e, 0x67, 0x65, 0x72, 0x32
  ]); // Hardcoded salt!
}
```

**Problems:**
- Hardcoded salt provides no real security
- Browser fingerprint is easily spoofable
- XSS can still read IndexedDB

**Recommendation:** 
- Use password-derived key for storage encryption
- Consider Web Crypto API's non-extractable keys
- Add Content Security Policy headers

#### 3. **Missing Input Validation (MEDIUM)**
**Location:** [`frontend/src/lib/crypto/ecc.ts:94-99`](frontend/src/lib/crypto/ecc.ts:94)

```typescript
export function fromHex(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2)
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16)  // No validation!
  }
  return bytes
}
```

**Problem:** No validation of hex string format or length.

**Fix:** Add validation:
```typescript
export function fromHex(hex: string): Uint8Array {
  if (!/^[0-9a-fA-F]*$/.test(hex)) {
    throw new Error('Invalid hex string');
  }
  if (hex.length % 2 !== 0) {
    throw new Error('Hex string must have even length');
  }
  // ... rest of code
}
```

#### 4. **Deprecated String Method**
**Location:** [`frontend/src/lib/crypto/ecc.ts:97`](frontend/src/lib/crypto/ecc.ts:97)

```typescript
bytes[i / 2] = parseInt(hex.substr(i, 2), 16)
```

**Issue:** `substr()` is deprecated. Use `substring()` or `slice()`.

---

## 🏗️ Architecture

### ✅ Strengths

1. **Clean Microservices Design**
   - Proper service separation (auth, messages, realtime, gateway)
   - gRPC for inter-service communication
   - Kafka for event-driven architecture

2. **Scalability Patterns**
   - Outbox pattern for reliable message delivery
   - Redis for distributed state
   - Kafka for async processing

3. **Good Error Handling**
   - Custom error types in each service
   - Proper error propagation

### ⚠️ Issues

#### 1. **Missing Rate Limiting (HIGH)**
**Location:** [`api-gateway/internal/cores/middleware/ratelimit.go`](api-gateway/internal/cores/middleware/ratelimit.go:1)

While rate limiting middleware exists, it's not clear if it's applied to all critical endpoints. WebSocket connections especially need rate limiting.

**Recommendation:**
- Apply rate limiting to login/register endpoints
- Limit WebSocket connection attempts
- Add per-user message rate limits

#### 2. **Incomplete TODO**
**Location:** [`message-service/internal/messaging/transport/grpc_handler.go:149`](message-service/internal/messaging/transport/grpc_handler.go:149)

```go
CompanionLogin: "", // TODO: Get from auth-service or store in chats table
```

**Impact:** Frontend may need companion login for UI display.

#### 3. **WebSocket Security**
**Location:** [`api-gateway/internal/gateway/transport/websocket.go:27-37`](api-gateway/internal/gateway/transport/websocket.go:27)

```go
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		if os.Getenv("ENV") == "development" {
			origin := r.Header.Get("Origin")
			return strings.HasPrefix(origin, "http://localhost:") || 
			       strings.HasPrefix(origin, "http://127.0.0.1:")
		}
		
		return true  // ⚠️ Accepts ALL origins in production!
	},
}
```

**Critical:** Production accepts all origins! This allows any website to connect.

**Fix:**
```go
CheckOrigin: func(r *http.Request) bool {
	if os.Getenv("ENV") == "development" {
		origin := r.Header.Get("Origin")
		return strings.HasPrefix(origin, "http://localhost:") || 
		       strings.HasPrefix(origin, "http://127.0.0.1:")
	}
	
	// Production: whitelist allowed origins
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	origin := r.Header.Get("Origin")
	for _, allowed := range allowedOrigins {
		if origin == strings.TrimSpace(allowed) {
			return true
		}
	}
	return false
},
```

---

## 🔄 Backend Services

### Auth Service

**Strengths:**
- Proper JWT token management
- Refresh token rotation
- Server-side salt generation

**Issues:**
1. **Password Change Security** [`auth-service/internal/auth/service/service.go:113-152`](auth-service/internal/auth/service/service.go:113)
   - Good: Validates old password
   - Good: Invalidates tokens after password change
   - Missing: Should notify user via email about password change
   - Missing: Should require re-authentication for sensitive operations

### Message Service

**Strengths:**
- Outbox pattern for reliability
- Proper message ownership validation
- Batch status updates

**Issues:**
1. **Chat ID Generation** [`message-service/internal/messaging/service/service.go:36-38`](message-service/internal/messaging/service/service.go:36)
   ```go
   ids := []string{senderID, recipientID}
   sort.Strings(ids)
   chatID = ids[0] + ":" + ids[1]
   ```
   - Good: Deterministic chat ID
   - Issue: Ignores provided `chatID` parameter
   - Could cause confusion if client sends different chatID

2. **Error Handling in Kafka Publishing** [`message-service/internal/messaging/service/service.go:60-82`](message-service/internal/messaging/service/service.go:60)
   - Good: Falls back to outbox on Kafka failure
   - Issue: Silent failure - user thinks message is sent but it's only queued
   - Should return different status or notify user

### Realtime Service

**Strengths:**
- Proper connection lifecycle management
- Heartbeat mechanism
- Graceful shutdown

**Issues:**
1. **Error Swallowing** [`realtime-service/internal/connection/service/hub.go:142-146`](realtime-service/internal/connection/service/hub.go:142)
   ```go
   func (h *Hub) DeliverMessage(ctx context.Context, msg *domain.Message) error {
       stream, err := h.GetStream(msg.RecipientID)
       if err != nil {
           return nil  // ⚠️ Returns nil on error!
       }
   ```
   - Should return error or log it
   - Message delivery failure is silently ignored

---

## 🎨 Frontend

### ✅ Strengths

1. **Clean Crypto Abstraction**
   - Well-organized crypto modules
   - Clear separation of concerns

2. **WebSocket Management**
   - Automatic reconnection
   - Exponential backoff
   - Proper cleanup

### ⚠️ Issues

#### 1. **Memory Leaks in WebSocket**
**Location:** [`frontend/src/lib/api/websocket.ts:102-105`](frontend/src/lib/api/websocket.ts:102)

```typescript
onMessage(handler: MessageHandler): () => void {
  this.messageHandlers.add(handler);
  return () => this.messageHandlers.delete(handler);
}
```

**Issue:** If components don't call the cleanup function, handlers accumulate.

**Recommendation:** Use WeakSet or implement automatic cleanup on disconnect.

#### 2. **Base64 Encoding Issues**
**Location:** [`frontend/src/lib/crypto/ecc.ts:105-107`](frontend/src/lib/crypto/ecc.ts:105)

```typescript
export function toBase64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
}
```

**Problem:** Spread operator can cause stack overflow for large arrays.

**Fix:**
```typescript
export function toBase64(bytes: Uint8Array): string {
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary);
}
```

#### 3. **Missing Error Boundaries**
No React error boundaries detected. Crypto errors could crash the entire app.

---

## 🐳 Infrastructure

### Docker Compose

**Strengths:**
- Health checks for all services
- Resource limits
- Proper network segmentation

**Issues:**

1. **Security Concerns:**
   - No secrets management (uses env vars)
   - Should use Docker secrets or external vault
   - Database passwords in plain text

2. **Missing Monitoring:**
   - No Prometheus/Grafana
   - No centralized logging (ELK/Loki)
   - No alerting

3. **Kafka Configuration:**
   - Single node (no HA)
   - Replication factor = 1
   - Not production-ready

---

## 📊 Performance

### Potential Bottlenecks

1. **Crypto Operations:**
   - 1M iterations in key derivation is slow (~1-2 seconds)
   - Consider Web Workers for crypto operations
   - Cache derived keys in memory

2. **Database Queries:**
   - No visible indexes in migrations
   - Should add indexes on:
     - `messages.chat_id`
     - `messages.created_at`
     - `chats.user_id`

3. **WebSocket Scaling:**
   - Hub stores connections in memory
   - Won't scale horizontally without sticky sessions
   - Consider Redis pub/sub for multi-instance support

---

## 🧪 Testing

**Missing:**
- No unit tests visible
- No integration tests
- No E2E tests
- No crypto test vectors

**Recommendation:** Add tests for:
- Crypto functions (with known test vectors)
- Key derivation edge cases
- Message encryption/decryption
- WebSocket reconnection logic

---

## 📝 Code Quality

### Good Practices

1. ✅ Consistent error handling
2. ✅ Good code organization
3. ✅ Meaningful variable names
4. ✅ Proper use of contexts in Go

### Issues

1. **Logging:**
   - Inconsistent log levels
   - Some errors logged as warnings
   - Missing structured logging in some places

2. **Comments:**
   - Good documentation in crypto modules
   - Missing in some backend services
   - No API documentation (Swagger/OpenAPI)

3. **Type Safety:**
   - TypeScript `any` types should be avoided
   - Some type assertions could be safer

---

## 🎯 Priority Recommendations

### Critical (Fix Immediately)

1. ✅ **Add salt to key derivation** - Prevents rainbow table attacks
2. ✅ **Fix WebSocket CORS in production** - Major security hole
3. ✅ **Add rate limiting** - Prevent abuse

### High Priority

4. ✅ **Improve storage encryption** - Better XSS protection
5. ✅ **Add input validation** - Prevent crashes
6. ✅ **Fix error handling in realtime service** - Message delivery reliability
7. ✅ **Add database indexes** - Performance

### Medium Priority

8. ✅ **Implement monitoring** - Observability
9. ✅ **Add tests** - Code quality
10. ✅ **Complete TODO items** - Feature completeness
11. ✅ **Add API documentation** - Developer experience

### Low Priority

12. ✅ **Refactor base64 encoding** - Edge case handling
13. ✅ **Add error boundaries** - Better UX
14. ✅ **Improve logging** - Debugging

---

## 🌟 Positive Highlights

1. **Solid cryptographic foundation** - Proper algorithms and modes
2. **Good architecture** - Microservices done right
3. **Scalability patterns** - Outbox, Kafka, Redis
4. **Clean code structure** - Easy to navigate
5. **Security-first mindset** - E2EE, no plaintext storage

---

## 📚 Additional Resources

- [OWASP Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)
- [Web Crypto API Best Practices](https://developer.mozilla.org/en-US/docs/Web/API/Web_Crypto_API)
- [Signal Protocol Documentation](https://signal.org/docs/)
- [NIST Password Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)

---

## Conclusion

This is a well-designed messenger with strong cryptographic foundations. The main concerns are:
- Key derivation needs salt
- WebSocket CORS misconfiguration
- Missing rate limiting
- Production readiness (monitoring, HA)

With the critical fixes applied, this would be a solid 8.5/10 project. The architecture is sound and the code quality is good. Focus on security hardening and production readiness.
