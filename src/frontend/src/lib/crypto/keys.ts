/**
 * Cryptographic key management
 * Generation, storage and restoration of keys
 */

import { derivePrivateKey, derivePublicKey, toHex } from './ecc'
import { sha256 } from '@noble/hashes/sha2.js'
import type { KeyPair } from '../../types/crypto'
import {
  savePrivateKeySecure,
  restorePrivateKeySecure,
  hasStoredKeySecure,
  clearKeysSecure,
  isPersistentStorageEnabled
} from './secureStorage'

/**
 * Generate key pair from password
 *
 * @param password - User password
 * @param iterations - Number of hashing iterations
 * @returns Key pair (private, public) and hash for authentication
 */
export async function generateKeyPair(
  password: string,
  iterations = 10000
): Promise<KeyPair> {
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
 * Save private key securely
 *
 * @param privateKey - Private key to save
 * @param rememberMe - If true, persist in IndexedDB (survives browser restart). If false, use sessionStorage (cleared on tab close).
 */
export async function savePrivateKey(
  privateKey: Uint8Array,
  rememberMe: boolean
): Promise<void> {
  await savePrivateKeySecure(privateKey, rememberMe)
}

/**
 * Restore private key from secure storage
 * Checks RAM first, then IndexedDB/sessionStorage
 * 
 * @returns Private key or null if not found
 */
export async function restorePrivateKey(): Promise<Uint8Array | null> {
  return await restorePrivateKeySecure()
}

/**
 * Check if key is stored
 * 
 * @returns true if key is stored
 */
export function hasStoredKey(): boolean {
  return hasStoredKeySecure()
}

/**
 * Clear all stored keys securely
 * Used on logout
 */
export async function clearKeys(): Promise<void> {
  await clearKeysSecure()
}

/**
 * Check if "Remember me" (persistent storage) is enabled
 * 
 * @returns true if user chose "Remember me"
 */
export function isRememberMeEnabled(): boolean {
  return isPersistentStorageEnabled()
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
