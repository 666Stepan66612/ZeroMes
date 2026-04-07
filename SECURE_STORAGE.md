# Secure Key Storage Implementation

## Overview

This implementation provides secure storage for private keys with protection against XSS attacks and proper session management.

## How It Works

### Storage Modes

**1. Session Storage (Remember Me = OFF)**
- Keys stored in `sessionStorage`
- Automatically cleared when browser tab is closed
- Survives page reloads within the same session
- **Use case**: Shared computers, public devices

**2. Persistent Storage (Remember Me = ON)**
- Keys stored in `IndexedDB` with encryption
- Survives browser restart and computer reboot
- Encrypted using browser fingerprint
- **Use case**: Personal devices

### Security Layers

#### Layer 1: Storage Location
- **sessionStorage**: Cleared on tab close (not accessible across tabs)
- **IndexedDB**: Persistent but isolated per origin

#### Layer 2: Encryption
When using persistent storage, keys are encrypted before storing:

```typescript
// Derive encryption key from browser fingerprint
const fingerprint = `${navigator.userAgent}|${navigator.language}|${screen.width}x${screen.height}`;
// Use PBKDF2 with 100,000 iterations
// Encrypt with AES-256-GCM
```

This means even if an XSS attack reads IndexedDB, the data is encrypted.

#### Layer 3: Memory Cache
Keys are cached in RAM (`window.encryptionKey`) for fast access during the session.

## API Usage

### Login
```typescript
import { generateKeyPair, savePrivateKey } from '@/lib/crypto';

// Generate keys from password
const keyPair = await generateKeyPair(password);

// Save with user's choice
await savePrivateKey(keyPair.privateKey, rememberMe);
// rememberMe = false: sessionStorage (cleared on tab close)
// rememberMe = true: IndexedDB (persists after reboot)
```

### Restore Key
```typescript
import { restorePrivateKey } from '@/lib/crypto';

// Automatically checks: RAM → sessionStorage → IndexedDB
const privateKey = await restorePrivateKey();
if (!privateKey) {
  // User needs to login
  navigate('/login');
}
```

### Logout
```typescript
import { clearKeys } from '@/lib/crypto';

// Clears all storage locations
await clearKeys();
// - Wipes RAM (fills with zeros)
// - Clears IndexedDB
// - Clears sessionStorage
// - Clears localStorage flags
```

## Behavior Matrix

| Scenario | Remember Me OFF | Remember Me ON |
|----------|----------------|----------------|
| Page reload | ✅ Key persists | ✅ Key persists |
| Close tab | ❌ Key cleared | ✅ Key persists |
| Close browser | ❌ Key cleared | ✅ Key persists |
| Reboot computer | ❌ Key cleared | ✅ Key persists |
| XSS attack reads storage | ⚠️ Exposed | 🔒 Encrypted |

## XSS Protection

### What We Protect Against
1. **Script injection reading localStorage**: Keys are in sessionStorage or encrypted in IndexedDB
2. **Script injection reading IndexedDB**: Keys are encrypted with browser-specific data
3. **Script injection reading memory**: Limited protection (XSS can still access `window` object)

### What We DON'T Protect Against
1. **Keylogger XSS**: If attacker injects script that captures password input
2. **Active session hijacking**: If XSS runs while user is logged in, it can access RAM cache
3. **Browser extension attacks**: Malicious extensions have full access

### Best Practices
To maximize security:

1. **Use Content-Security-Policy (CSP)** headers to prevent XSS
2. **Sanitize all user input** before rendering
3. **Use HTTPS** to prevent MITM attacks
4. **Regular security audits** of dependencies
5. **Educate users** about phishing and malicious extensions

## Migration from Old System

### Old Behavior (WRONG)
```typescript
// Old: Always used localStorage
localStorage.setItem('private_key', base64Key);
localStorage.setItem('remember_me', rememberMe ? 'true' : 'false');
```

Problems:
- localStorage persists forever (even without "Remember Me")
- No encryption
- Vulnerable to XSS

### New Behavior (CORRECT)
```typescript
// New: sessionStorage OR encrypted IndexedDB
if (rememberMe) {
  // IndexedDB with encryption
  await saveToIndexedDB(encryptedKey);
} else {
  // sessionStorage (cleared on tab close)
  sessionStorage.setItem('private_key', base64Key);
}
```

## Implementation Details

### Files Modified
- `frontend/src/lib/crypto/secureStorage.ts` - New secure storage implementation
- `frontend/src/lib/crypto/keys.ts` - Updated to use secure storage
- `frontend/src/pages/LoginPage.tsx` - Updated to async functions
- `frontend/src/pages/ChatsPage.tsx` - Updated to async functions
- `frontend/src/pages/ChangePasswordPage.tsx` - Updated to async functions
- `frontend/src/components/ChatWindow.tsx` - Updated to async functions

### Breaking Changes
All key management functions are now async:

```typescript
// Before
const key = restorePrivateKey();
clearKeys();
savePrivateKey(key, true);

// After
const key = await restorePrivateKey();
await clearKeys();
await savePrivateKey(key, true);
```

## Testing

### Test Session Storage (Remember Me OFF)
1. Login without checking "Remember Me"
2. Reload page → Should stay logged in ✅
3. Close tab and reopen → Should require login ✅

### Test Persistent Storage (Remember Me ON)
1. Login with "Remember Me" checked
2. Reload page → Should stay logged in ✅
3. Close browser and reopen → Should stay logged in ✅
4. Reboot computer → Should stay logged in ✅

### Test Logout
1. Login (either mode)
2. Click logout
3. Check DevTools:
   - sessionStorage should be empty
   - IndexedDB should have no keys
   - localStorage should have no private_key

## Future Improvements

1. **Web Crypto API Key Storage**: Use `crypto.subtle.generateKey()` with `extractable: false`
2. **Hardware Security Keys**: Support WebAuthn for key storage
3. **Biometric Authentication**: Use Web Authentication API
4. **Key Rotation**: Automatically rotate keys periodically
5. **Multi-device Sync**: Encrypted key backup to server

## Security Considerations

### Threat Model
- ✅ Protected: Passive XSS reading storage
- ✅ Protected: Storage persistence without user consent
- ⚠️ Partial: Active XSS during session
- ❌ Not Protected: Keylogger, MITM without HTTPS, malicious extensions

### Recommendations
1. **Always use HTTPS** in production
2. **Implement CSP** headers (already in Caddyfile)
3. **Regular dependency updates** to patch vulnerabilities
4. **User education** about device security
5. **Consider adding** 2FA for additional security layer

## Conclusion

This implementation provides a good balance between security and usability:
- Users without "Remember Me" get automatic logout on tab close
- Users with "Remember Me" get persistent login with encrypted storage
- XSS attacks have limited access to encrypted data
- Clear separation between session and persistent storage

The system is not perfect (no client-side storage is 100% secure against XSS), but it's significantly better than storing unencrypted keys in localStorage.
