/**
 * Типы для криптографических операций
 */

export interface KeyPair {
  privateKey: Uint8Array  // 32 байта - приватный ключ
  publicKey: Uint8Array   // 65 байт (несжатый) или 33 байта (сжатый)
  authHash: string        // Hex string для аутентификации на сервере
}

export interface EncryptedMessage {
  ciphertext: string  // Base64 зашифрованное сообщение
  nonce: string       // Base64 nonce для AES-GCM
}

export interface ChatKeyInfo {
  chatId: string
  companionId: string
  sharedKey: Uint8Array  // 32 байта - ключ для шифрования сообщений
}
