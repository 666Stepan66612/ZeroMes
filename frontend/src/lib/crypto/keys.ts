/**
 * Cryptographic key management
 * Generation, storage and restoration of keys
 */

import { derivePrivateKey, derivePublicKey, toBase64, fromBase64, toHex } from './ecc'
import { sha256 } from '@noble/hashes/sha2.js'
import type { KeyPair } from '../../types/crypto'

/**
 * Generate key pair from password
 * 
 * @param password - User password
 * @param iterations - Number of hashing iterations
 * @returns Key pair (private, public) and hash for authentication
 */
export function generateKeyPair(
  password: string,
  iterations = 10000
): KeyPair {
  // 1. Generate private key from password
  const privateKey = derivePrivateKey(password, iterations)
  
  // 2. Generate public key (uncompressed format - 65 bytes)
  const publicKey = derivePublicKey(privateKey, false)
  
  // 3. Generate hash for authentication (separate context)
  let authHashBytes: Uint8Array = new TextEncoder().encode(password + 'auth')
  for (let i = 0; i < iterations; i++) {
    authHashBytes = new Uint8Array(sha256(authHashBytes))
  }
  
  return {
    privateKey,
    publicKey,
    authHash: toHex(authHashBytes)
  }
}

/**
 * Save private key
 * 
 * @param privateKey - Private key to save
 * @param rememberMe - Save to localStorage (true) or RAM only (false)
 */
export function savePrivateKey(
  privateKey: Uint8Array,
  rememberMe: boolean
): void {
  const base64Key = toBase64(privateKey)
  
  if (rememberMe) {
    // Save to localStorage (on disk, but deleted when browser cache is cleared)
    localStorage.setItem('private_key', base64Key)
    localStorage.setItem('remember_me', 'true')
    console.log('🔓 Key saved to localStorage')
  } else {
    // Save to RAM only (deleted when tab is closed)
    ;(window as any).encryptionKey = privateKey
    console.log('🔒 Key in memory only (more secure)')
  }
}

/**
 * Restore private key
 * Checks RAM first, then localStorage
 * 
 * @returns Private key or null if not found
 */
export function restorePrivateKey(): Uint8Array | null {
  // Check RAM first
  const ramKey = (window as any).encryptionKey
  if (ramKey instanceof Uint8Array) {
    return ramKey
  }
  
  // Then check localStorage
  const stored = localStorage.getItem('private_key')
  if (stored) {
    try {
      return fromBase64(stored)
    } catch (error) {
      console.error('Error restoring key from localStorage:', error)
      // Clear corrupted key
      localStorage.removeItem('private_key')
      return null
    }
  }
  
  return null
}

/**
 * Check if key is stored
 * 
 * @returns true if key is stored
 */
export function hasStoredKey(): boolean {
  return !!(
    (window as any).encryptionKey ||
    localStorage.getItem('private_key')
  )
}

/**
 * Clear all stored keys
 * Used on logout
 */
export function clearKeys(): void {
  // Clear RAM
  if ((window as any).encryptionKey) {
    // Wipe memory (fill with zeros)
    const key = (window as any).encryptionKey
    if (key instanceof Uint8Array) {
      key.fill(0)
    }
    delete (window as any).encryptionKey
  }
  
  // Clear localStorage
  localStorage.removeItem('private_key')
  localStorage.removeItem('remember_me')
  
  console.log('✅ All keys deleted')
}

/**
 * Check if "Remember me" flag is set
 * 
 * @returns true if user chose "Remember me"
 */
export function isRememberMeEnabled(): boolean {
  return localStorage.getItem('remember_me') === 'true'
}

/**
 * Restore public key from private key
 * Useful when you need public key without re-entering password
 * 
 * @param privateKey - Private key
 * @returns Public key
 */
export function getPublicKeyFromPrivate(privateKey: Uint8Array): Uint8Array {
  return derivePublicKey(privateKey, false)
}
