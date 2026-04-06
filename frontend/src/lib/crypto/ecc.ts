/**
 * Elliptic curve cryptography on secp256k1 curve
 * Used for ECDH (Elliptic Curve Diffie-Hellman)
 */

import * as secp256k1 from '@noble/secp256k1'
import { sha256 } from '@noble/hashes/sha2.js'

/**
 * Generate private key from password
 * Uses multiple SHA256 iterations for strengthening
 * 
 * @param password - User password
 * @param iterations - Number of hashing iterations (default 10000)
 * @returns Private key (32 bytes)
 */
export function derivePrivateKey(password: string, iterations = 10000): Uint8Array {
  let hash: Uint8Array = new TextEncoder().encode(password)
  
  // Multiple SHA256 iterations for brute-force protection
  for (let i = 0; i < iterations; i++) {
    hash = new Uint8Array(sha256(hash))
  }
  
  return hash  // 32 bytes
}

/**
 * Generate public key from private key
 * Uses multiplication by base point G of secp256k1 curve
 * 
 * @param privateKey - Private key (32 bytes)
 * @param compressed - Use compressed format (33 bytes instead of 65)
 * @returns Public key
 */
export function derivePublicKey(privateKey: Uint8Array, compressed = false): Uint8Array {
  return secp256k1.getPublicKey(privateKey, compressed)
}

/**
 * ECDH: Compute shared secret between two parties
 * Alice: sharedSecret = alicePrivateKey * bobPublicKey
 * Bob: sharedSecret = bobPrivateKey * alicePublicKey
 * Result is the same due to elliptic curve properties
 * 
 * @param privateKey - My private key
 * @param publicKey - Companion's public key
 * @returns Shared secret (32 bytes) for use in AES
 */
export function computeSharedSecret(
  privateKey: Uint8Array,
  publicKey: Uint8Array
): Uint8Array {
  // Get shared point on curve
  const sharedPoint = secp256k1.getSharedSecret(privateKey, publicKey)
  
  // Take x coordinate (skip first byte - prefix)
  // sharedPoint format: [prefix, x(32 bytes), y(32 bytes)]
  const x = sharedPoint.slice(1, 33)
  
  // Hash x coordinate to get encryption key
  return new Uint8Array(sha256(x))  // 32 bytes for AES-256
}

/**
 * Validate public key
 * 
 * @param publicKey - Public key to validate
 * @returns true if key is valid
 */
export function isValidPublicKey(publicKey: Uint8Array): boolean {
  try {
    // Convert to hex for validation
    const hex = toHex(publicKey)
    secp256k1.Point.fromHex(hex)
    return true
  } catch {
    return false
  }
}

/**
 * Convert Uint8Array to hex string
 */
export function toHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
}

/**
 * Convert hex string to Uint8Array
 */
export function fromHex(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2)
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16)
  }
  return bytes
}

/**
 * Convert Uint8Array to base64
 */
export function toBase64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
}

/**
 * Convert base64 to Uint8Array
 */
export function fromBase64(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}
