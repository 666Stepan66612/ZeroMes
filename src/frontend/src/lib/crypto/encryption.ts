/**
 * Message encryption and decryption
 * Uses AES-256-GCM for symmetric encryption
 */

import { computeSharedSecret } from './ecc'
import { getCryptoSubtle } from './utils'
import type { EncryptedMessage } from '../../types/crypto'

/**
 * Encrypt message using AES-256-GCM
 * 
 * @param message - Message text to encrypt
 * @param chatKey - Encryption key (32 bytes)
 * @returns Encrypted message and nonce
 */
export async function encryptMessage(
  message: string,
  chatKey: Uint8Array
): Promise<EncryptedMessage> {
  const subtle = getCryptoSubtle();
  
  // Generate nonce (12 bytes for GCM)
  const nonce = crypto.getRandomValues(new Uint8Array(12))
  
  // Import key for Web Crypto API
  const key = await subtle.importKey(
    'raw',
    chatKey as BufferSource,
    { name: 'AES-GCM' },
    false,
    ['encrypt']
  )
  
  // Encode message to bytes
  const messageBytes = new TextEncoder().encode(message)
  
  // Encrypt
  const ciphertext = await subtle.encrypt(
    { name: 'AES-GCM', iv: nonce },
    key,
    messageBytes
  )
  
  return {
    ciphertext: arrayBufferToBase64(ciphertext),
    nonce: arrayBufferToBase64(nonce.buffer)
  }
}

/**
 * Decrypt message
 * 
 * @param ciphertext - Encrypted message (base64)
 * @param nonce - Nonce used during encryption (base64)
 * @param chatKey - Encryption key (32 bytes)
 * @returns Decrypted message text
 * @throws Error if decryption fails
 */
export async function decryptMessage(
  ciphertext: string,
  nonce: string,
  chatKey: Uint8Array
): Promise<string> {
  const subtle = getCryptoSubtle();
  
  // Import key
  const key = await subtle.importKey(
    'raw',
    chatKey as BufferSource,
    { name: 'AES-GCM' },
    false,
    ['decrypt']
  )
  
  // Convert from base64
  const ciphertextBytes = base64ToArrayBuffer(ciphertext)
  const nonceBytes = base64ToArrayBuffer(nonce)
  
  try {
    // Decryption
    const plaintext = await subtle.decrypt(
      { name: 'AES-GCM', iv: nonceBytes as BufferSource },
      key,
      ciphertextBytes as BufferSource
    )
    
    // Decode to string
    return new TextDecoder().decode(plaintext)
  } catch (error) {
    console.error('Decryption error:', error)
    throw new Error('Failed to decrypt message. Key might be incorrect.')
  }
}

/**
 * Derive chat key from ECDH
 * Computes shared secret between two users
 * 
 * @param myPrivateKey - My private key
 * @param theirPublicKey - Their public key
 * @returns Key for encrypting messages in chat (32 bytes)
 */
export function deriveChatKey(
  myPrivateKey: Uint8Array,
  theirPublicKey: Uint8Array
): Uint8Array {
  return computeSharedSecret(myPrivateKey, theirPublicKey)
}

/**
 * Convert ArrayBuffer to base64
 */
function arrayBufferToBase64(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary)
}

/**
 * Convert base64 to ArrayBuffer
 */
function base64ToArrayBuffer(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}

/**
 * Generate random key for new chat
 * Use if you don't want to use ECDH
 * 
 * @returns Random key (32 bytes)
 */
export function generateRandomKey(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(32))
}

/**
 * Encrypt chat key with user's public key
 * Used to save chat key on server
 * 
 * @param chatKey - Chat key to encrypt
 * @param password - User password for encryption
 * @returns Encrypted key
 */
/**
 * Encrypt chat key with private key (for storage)
 * Uses private key directly as AES key
 */
export async function encryptChatKeyWithPrivateKey(
  chatKey: Uint8Array,
  privateKey: Uint8Array
): Promise<EncryptedMessage> {
  const subtle = getCryptoSubtle();
  
  // Use private key as AES key (32 bytes)
  const aesKey = await subtle.importKey(
    'raw',
    privateKey as BufferSource,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt']
  )
  
  const nonce = crypto.getRandomValues(new Uint8Array(12))
  
  const encrypted = await subtle.encrypt(
    { name: 'AES-GCM', iv: nonce },
    aesKey,
    chatKey as BufferSource
  )
  
  // Combine nonce + encrypted
  const combined = new Uint8Array(nonce.length + encrypted.byteLength)
  combined.set(nonce, 0)
  combined.set(new Uint8Array(encrypted), nonce.length)
  
  return {
    ciphertext: arrayBufferToBase64(combined.buffer),
    nonce: '' // Not used, nonce is included in ciphertext
  }
}

/**
 * Decrypt chat key using private key
 *
 * @param encryptedKey - Encrypted chat key (base64)
 * @param privateKey - User's private key (32 bytes)
 * @returns Decrypted chat key
 */
export async function decryptChatKeyWithPrivateKey(
  encryptedKey: string,
  privateKey: Uint8Array
): Promise<Uint8Array> {
  const subtle = getCryptoSubtle();
  const combined = base64ToArrayBuffer(encryptedKey)
  
  // Extract nonce and encrypted data
  const nonce = combined.slice(0, 12)
  const encrypted = combined.slice(12)
  
  // Use private key as AES key
  const aesKey = await subtle.importKey(
    'raw',
    privateKey as BufferSource,
    { name: 'AES-GCM', length: 256 },
    false,
    ['decrypt']
  )
  
  const decrypted = await subtle.decrypt(
    { name: 'AES-GCM', iv: nonce },
    aesKey,
    encrypted
  )
  
  return new Uint8Array(decrypted)
}

export async function encryptChatKey(
  chatKey: Uint8Array,
  password: string
): Promise<EncryptedMessage> {
  // Derive key from password to encrypt chat key
  const encoder = new TextEncoder()
  const passwordBytes = encoder.encode(password)
  
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    passwordBytes,
    { name: 'PBKDF2' },
    false,
    ['deriveBits', 'deriveKey']
  )
  
  const salt = crypto.getRandomValues(new Uint8Array(16))
  
  const wrappingKey = await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt']
  )
  
  const nonce = crypto.getRandomValues(new Uint8Array(12))
  
  const encrypted = await crypto.subtle.encrypt(
    { name: 'AES-GCM', iv: nonce },
    wrappingKey,
    chatKey as BufferSource
  )
  
  // Combine salt + nonce + encrypted
  const combined = new Uint8Array(salt.length + nonce.length + encrypted.byteLength)
  combined.set(salt, 0)
  combined.set(nonce, salt.length)
  combined.set(new Uint8Array(encrypted), salt.length + nonce.length)
  
  return {
    ciphertext: arrayBufferToBase64(combined.buffer),
    nonce: '' // Not used, nonce is included in ciphertext
  }
}

/**
 * Decrypt chat key
 * 
 * @param encryptedKey - Encrypted chat key
 * @param password - User password
 * @returns Decrypted chat key
 */
export async function decryptChatKey(
  encryptedKey: string,
  password: string
): Promise<Uint8Array> {
  const combined = base64ToArrayBuffer(encryptedKey)
  
  // Extract salt, nonce and encrypted
  const salt = combined.slice(0, 16)
  const nonce = combined.slice(16, 28)
  const encrypted = combined.slice(28)
  
  const encoder = new TextEncoder()
  const passwordBytes = encoder.encode(password)
  
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    passwordBytes,
    { name: 'PBKDF2' },
    false,
    ['deriveBits', 'deriveKey']
  )
  
  const wrappingKey = await crypto.subtle.deriveKey(
    {
      name: 'PBKDF2',
      salt: salt,
      iterations: 100000,
      hash: 'SHA-256'
    },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['decrypt']
  )
  
  const decrypted = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: nonce },
    wrappingKey,
    encrypted
  )
  
  return new Uint8Array(decrypted)
}
