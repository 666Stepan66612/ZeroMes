/**
 * Tests for encryption module
 */

import { describe, it, expect, beforeAll } from 'vitest'
import {
  encryptMessage,
  decryptMessage,
  deriveChatKey,
  generateRandomKey,
  encryptChatKeyWithPrivateKey,
  decryptChatKeyWithPrivateKey,
  encryptChatKey,
  decryptChatKey,
} from '../encryption'
import { generateKeyPair } from '../ecc'

describe('Message Encryption', () => {
  let chatKey: Uint8Array

  beforeAll(() => {
    chatKey = generateRandomKey()
  })

  it('should encrypt and decrypt a message', async () => {
    const message = 'Hello, World!'

    const encrypted = await encryptMessage(message, chatKey)

    expect(encrypted.ciphertext).toBeTruthy()
    expect(encrypted.nonce).toBeTruthy()

    const decrypted = await decryptMessage(
      encrypted.ciphertext,
      encrypted.nonce,
      chatKey
    )

    expect(decrypted).toBe(message)
  })

  it('should encrypt and decrypt unicode messages', async () => {
    const message = 'Привет, мир! 🌍 こんにちは'

    const encrypted = await encryptMessage(message, chatKey)
    const decrypted = await decryptMessage(
      encrypted.ciphertext,
      encrypted.nonce,
      chatKey
    )

    expect(decrypted).toBe(message)
  })

  it('should fail to decrypt with wrong key', async () => {
    const message = 'Secret message'
    const wrongKey = generateRandomKey()

    const encrypted = await encryptMessage(message, chatKey)

    await expect(
      decryptMessage(encrypted.ciphertext, encrypted.nonce, wrongKey)
    ).rejects.toThrow()
  })

  it('should produce different ciphertext for same message', async () => {
    const message = 'Same message'

    const encrypted1 = await encryptMessage(message, chatKey)
    const encrypted2 = await encryptMessage(message, chatKey)

    // Different nonces should produce different ciphertexts
    expect(encrypted1.ciphertext).not.toBe(encrypted2.ciphertext)
    expect(encrypted1.nonce).not.toBe(encrypted2.nonce)

    // But both should decrypt to the same message
    const decrypted1 = await decryptMessage(
      encrypted1.ciphertext,
      encrypted1.nonce,
      chatKey
    )
    const decrypted2 = await decryptMessage(
      encrypted2.ciphertext,
      encrypted2.nonce,
      chatKey
    )

    expect(decrypted1).toBe(message)
    expect(decrypted2).toBe(message)
  })

  it('should handle empty messages', async () => {
    const message = ''

    const encrypted = await encryptMessage(message, chatKey)
    const decrypted = await decryptMessage(
      encrypted.ciphertext,
      encrypted.nonce,
      chatKey
    )

    expect(decrypted).toBe(message)
  })

  it('should handle long messages', async () => {
    const message = 'A'.repeat(10000)

    const encrypted = await encryptMessage(message, chatKey)
    const decrypted = await decryptMessage(
      encrypted.ciphertext,
      encrypted.nonce,
      chatKey
    )

    expect(decrypted).toBe(message)
  })
})

describe('Key Derivation', () => {
  it('should derive same chat key from ECDH', () => {
    const alice = generateKeyPair()
    const bob = generateKeyPair()

    // Alice derives key using her private key and Bob's public key
    const aliceKey = deriveChatKey(alice.privateKey, bob.publicKey)

    // Bob derives key using his private key and Alice's public key
    const bobKey = deriveChatKey(bob.privateKey, alice.publicKey)

    // Both should get the same key
    expect(aliceKey).toEqual(bobKey)
  })

  it('should generate random keys of correct length', () => {
    const key1 = generateRandomKey()
    const key2 = generateRandomKey()

    expect(key1.length).toBe(32)
    expect(key2.length).toBe(32)
    expect(key1).not.toEqual(key2)
  })
})

describe('Chat Key Encryption with Private Key', () => {
  it('should encrypt and decrypt chat key with private key', async () => {
    const chatKey = generateRandomKey()
    const { privateKey } = generateKeyPair()

    const encrypted = await encryptChatKeyWithPrivateKey(chatKey, privateKey)

    expect(encrypted.ciphertext).toBeTruthy()

    const decrypted = await decryptChatKeyWithPrivateKey(
      encrypted.ciphertext,
      privateKey
    )

    expect(decrypted).toEqual(chatKey)
  })

  it('should fail to decrypt with wrong private key', async () => {
    const chatKey = generateRandomKey()
    const { privateKey: key1 } = generateKeyPair()
    const { privateKey: key2 } = generateKeyPair()

    const encrypted = await encryptChatKeyWithPrivateKey(chatKey, key1)

    await expect(
      decryptChatKeyWithPrivateKey(encrypted.ciphertext, key2)
    ).rejects.toThrow()
  })
})

describe('Chat Key Encryption with Password', () => {
  it('should encrypt and decrypt chat key with password', async () => {
    const chatKey = generateRandomKey()
    const password = 'strong-password-123'

    const encrypted = await encryptChatKey(chatKey, password)

    expect(encrypted.ciphertext).toBeTruthy()

    const decrypted = await decryptChatKey(encrypted.ciphertext, password)

    expect(decrypted).toEqual(chatKey)
  })

  it('should fail to decrypt with wrong password', async () => {
    const chatKey = generateRandomKey()
    const password = 'correct-password'
    const wrongPassword = 'wrong-password'

    const encrypted = await encryptChatKey(chatKey, password)

    await expect(
      decryptChatKey(encrypted.ciphertext, wrongPassword)
    ).rejects.toThrow()
  })

  it('should produce different ciphertext with same password', async () => {
    const chatKey = generateRandomKey()
    const password = 'same-password'

    const encrypted1 = await encryptChatKey(chatKey, password)
    const encrypted2 = await encryptChatKey(chatKey, password)

    // Different salts should produce different ciphertexts
    expect(encrypted1.ciphertext).not.toBe(encrypted2.ciphertext)

    // But both should decrypt to the same key
    const decrypted1 = await decryptChatKey(encrypted1.ciphertext, password)
    const decrypted2 = await decryptChatKey(encrypted2.ciphertext, password)

    expect(decrypted1).toEqual(chatKey)
    expect(decrypted2).toEqual(chatKey)
  })
})
