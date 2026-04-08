/**
 * Cryptographic module
 * Export all encryption functions
 */

// Utility functions
export {
  isCryptoAvailable,
  getCryptoSubtle,
  isSecureContext
} from './utils'

// ECC functions
export {
  derivePrivateKey,
  derivePublicKey,
  computeSharedSecret,
  isValidPublicKey,
  toHex,
  fromHex,
  toBase64,
  fromBase64
} from './ecc'

// Key management
export {
  generateKeyPair,
  savePrivateKey,
  restorePrivateKey,
  hasStoredKey,
  clearKeys,
  isRememberMeEnabled,
  getPublicKeyFromPrivate
} from './keys'

// Encryption
export {
  encryptMessage,
  decryptMessage,
  deriveChatKey,
  generateRandomKey,
  encryptChatKey,
  encryptChatKeyWithPrivateKey,
  decryptChatKey
} from './encryption'

// Types
export type { KeyPair, EncryptedMessage, ChatKeyInfo } from '../../types/crypto'
