/**
 * Types for cryptographic operations
 */

export interface KeyPair {
  privateKey: Uint8Array  // 32 bytes - private key
  publicKey: Uint8Array   // 65 bytes (uncompressed) or 33 bytes (compressed)
  authHash: string        // Hex string for server authentication
}

export interface EncryptedMessage {
  ciphertext: string  // Base64 encrypted message
  nonce: string       // Base64 nonce for AES-GCM
}

export interface ChatKeyInfo {
  chatId: string
  companionId: string
  sharedKey: Uint8Array  // 32 bytes - key for message encryption
}
