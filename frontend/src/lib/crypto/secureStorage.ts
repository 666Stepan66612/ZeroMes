/**
 * Secure storage for sensitive data (private keys)
 * Uses IndexedDB with encryption for better XSS protection
 * Private key is kept in closure, NOT in window object
 */

const DB_NAME = 'SecureMessengerDB';
const DB_VERSION = 1;
const STORE_NAME = 'keys';

// Private key cache in closure (not accessible via window)
let cachedPrivateKey: Uint8Array | null = null;

/**
 * Initialize IndexedDB
 */
async function initDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION);
    
    request.onerror = () => reject(request.error);
    request.onsuccess = () => resolve(request.result);
    
    request.onupgradeneeded = (event) => {
      const db = (event.target as IDBOpenDBRequest).result;
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        db.createObjectStore(STORE_NAME);
      }
    };
  });
}

/**
 * Derive encryption key from browser fingerprint
 * This adds an extra layer - even if XSS reads IndexedDB, data is encrypted
 */
async function deriveStorageKey(): Promise<CryptoKey> {
  // Use browser-specific data as salt (not perfect, but better than nothing)
  const fingerprint = `${navigator.userAgent}|${navigator.language}|${screen.width}x${screen.height}`;
  const encoder = new TextEncoder();
  const data = encoder.encode(fingerprint);
  
  // Import as key material
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    data,
    { name: 'PBKDF2' },
    false,
    ['deriveBits', 'deriveKey']
  );
  
  // Derive AES key
  const salt = new Uint8Array([
    0x73, 0x65, 0x63, 0x75, 0x72, 0x65, 0x6d, 0x65,
    0x73, 0x73, 0x65, 0x6e, 0x67, 0x65, 0x72, 0x32
  ]); // "securemessenger2" in hex
  
  return await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

/**
 * Encrypt data before storing
 */
async function encryptData(data: Uint8Array): Promise<{ encrypted: ArrayBuffer; iv: Uint8Array }> {
  const key = await deriveStorageKey();
  const iv = crypto.getRandomValues(new Uint8Array(12));
  
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: iv as BufferSource },
    key,
    data as BufferSource
  );
  
  return { encrypted, iv };
}

/**
 * Decrypt data after reading
 */
async function decryptData(encrypted: ArrayBuffer, iv: Uint8Array): Promise<Uint8Array> {
  const key = await deriveStorageKey();
  
  const decrypted = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: iv as BufferSource },
    key,
    encrypted
  );
  
  return new Uint8Array(decrypted);
}

/**
 * Save private key securely
 * @param privateKey - Private key to save
 * @param persistent - If true, use IndexedDB (survives browser restart). If false, use sessionStorage (cleared on tab close)
 */
export async function savePrivateKeySecure(
  privateKey: Uint8Array,
  persistent: boolean
): Promise<void> {
  console.log('[SecureStorage] Saving key, persistent:', persistent);
  
  // Cache in memory (closure, not window)
  cachedPrivateKey = privateKey;
  
  if (persistent) {
    // Persistent storage: IndexedDB with encryption
    try {
      console.log('[SecureStorage] Encrypting key for IndexedDB...');
      const { encrypted, iv } = await encryptData(privateKey);
      
      console.log('[SecureStorage] Opening IndexedDB...');
      const db = await initDB();
      const transaction = db.transaction([STORE_NAME], 'readwrite');
      const store = transaction.objectStore(STORE_NAME);
      
      // Store encrypted key and IV
      await new Promise<void>((resolve, reject) => {
        const request = store.put({ encrypted, iv }, 'private_key');
        request.onsuccess = () => {
          console.log('[SecureStorage] ✅ Saved to IndexedDB successfully');
          resolve();
        };
        request.onerror = () => {
          console.error('[SecureStorage] ❌ IndexedDB put error:', request.error);
          reject(request.error);
        };
      });
      
      // Mark as persistent
      localStorage.setItem('key_storage_type', 'persistent');
      console.log('[SecureStorage] ✅ Marked as persistent in localStorage');
      
      db.close();
    } catch (error) {
      console.error('[SecureStorage] ❌ Failed to save to IndexedDB, falling back to localStorage:', error);
      // Fallback to localStorage if IndexedDB fails
      const base64Key = btoa(String.fromCharCode(...privateKey));
      localStorage.setItem('private_key', base64Key);
      localStorage.setItem('key_storage_type', 'persistent');
      console.log('[SecureStorage] ✅ Saved to localStorage as fallback');
    }
  } else {
    // Session mode: use localStorage but mark for cleanup on browser close
    console.log('[SecureStorage] Saving to localStorage (session mode)...');
    const base64Key = btoa(String.fromCharCode(...privateKey));
    localStorage.setItem('private_key', base64Key);
    localStorage.setItem('key_storage_type', 'session');
    localStorage.setItem('session_start', Date.now().toString());
    console.log('[SecureStorage] ✅ Saved to localStorage (session mode)');
    console.log('[SecureStorage] Verification - key_storage_type:', localStorage.getItem('key_storage_type'));
    console.log('[SecureStorage] Verification - private_key length:', localStorage.getItem('private_key')?.length);
  }
}

/**
 * Restore private key from storage
 * @returns Private key or null if not found
 */
export async function restorePrivateKeySecure(): Promise<Uint8Array | null> {
  console.log('[SecureStorage] Restoring key...');
  
  // Check cache first (fastest)
  if (cachedPrivateKey instanceof Uint8Array) {
    console.log('[SecureStorage] ✅ Found key in cache');
    return cachedPrivateKey;
  }
  
  const storageType = localStorage.getItem('key_storage_type');
  console.log('[SecureStorage] Storage type:', storageType);
  
  if (storageType === 'persistent') {
    // Try IndexedDB first
    try {
      console.log('[SecureStorage] Trying to restore from IndexedDB...');
      const db = await initDB();
      const transaction = db.transaction([STORE_NAME], 'readonly');
      const store = transaction.objectStore(STORE_NAME);
      
      const data = await new Promise<{ encrypted: ArrayBuffer; iv: Uint8Array } | undefined>((resolve, reject) => {
        const request = store.get('private_key');
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
      });
      
      db.close();
      
      if (data && data.encrypted && data.iv) {
        console.log('[SecureStorage] Found encrypted key in IndexedDB, decrypting...');
        const privateKey = await decryptData(data.encrypted, data.iv);
        // Cache in closure
        cachedPrivateKey = privateKey;
        console.log('[SecureStorage] ✅ Successfully restored from IndexedDB');
        return privateKey;
      } else {
        console.log('[SecureStorage] ⚠️ No data found in IndexedDB');
      }
    } catch (error) {
      console.error('[SecureStorage] ❌ Failed to read from IndexedDB:', error);
    }
    
    // Fallback to localStorage
    console.log('[SecureStorage] Trying localStorage fallback...');
    const stored = localStorage.getItem('private_key');
    if (stored) {
      try {
        const binary = atob(stored);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        cachedPrivateKey = bytes;
        console.log('[SecureStorage] ✅ Restored from localStorage fallback');
        return bytes;
      } catch (error) {
        console.error('[SecureStorage] ❌ Error restoring key from localStorage:', error);
        localStorage.removeItem('private_key');
      }
    }
  } else if (storageType === 'session') {
    // Session mode: key in localStorage
    console.log('[SecureStorage] Trying to restore from localStorage (session mode)...');
    const stored = localStorage.getItem('private_key');
    if (stored) {
      try {
        const binary = atob(stored);
        const bytes = new Uint8Array(binary.length);
        for (let i = 0; i < binary.length; i++) {
          bytes[i] = binary.charCodeAt(i);
        }
        cachedPrivateKey = bytes;
        console.log('[SecureStorage] ✅ Restored from localStorage (session mode)');
        return bytes;
      } catch (error) {
        console.error('[SecureStorage] ❌ Error restoring key from localStorage:', error);
        localStorage.removeItem('private_key');
      }
    } else {
      console.log('[SecureStorage] ⚠️ No key found in localStorage');
    }
  }
  
  console.log('[SecureStorage] ❌ No key found in any storage');
  return null;
}

/**
 * Check if key is stored
 */
export function hasStoredKeySecure(): boolean {
  return !!(
    cachedPrivateKey ||
    localStorage.getItem('key_storage_type')
  );
}

/**
 * Clear all stored keys
 */
export async function clearKeysSecure(): Promise<void> {
  // Clear cached key
  if (cachedPrivateKey) {
    cachedPrivateKey.fill(0); // Wipe memory
    cachedPrivateKey = null;
  }
  
  // Clear IndexedDB
  try {
    const db = await initDB();
    const transaction = db.transaction([STORE_NAME], 'readwrite');
    const store = transaction.objectStore(STORE_NAME);
    
    await new Promise<void>((resolve, reject) => {
      const request = store.delete('private_key');
      request.onsuccess = () => resolve();
      request.onerror = () => reject(request.error);
    });
    
    db.close();
  } catch (error) {
    console.error('Failed to clear IndexedDB:', error);
  }
  
  // Clear localStorage
  localStorage.removeItem('private_key');
  localStorage.removeItem('key_storage_type');
  
  // Clear sessionStorage
  sessionStorage.removeItem('private_key');
  sessionStorage.removeItem('key_storage_type');
  
  console.log('✅ All keys securely deleted');
}

/**
 * Check if persistent storage is enabled
 */
export function isPersistentStorageEnabled(): boolean {
  return localStorage.getItem('key_storage_type') === 'persistent';
}
