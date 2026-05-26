/**
 * Tests for keys module
 */

import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import {
  generateKeyPair,
  savePrivateKey,
  restorePrivateKey,
  hasStoredKey,
  clearKeys,
  getPublicKeyFromPrivate,
} from '../keys'

describe('Key Pair Generation', () => {
  it('should generate key pair from password', async () => {
    const password = 'test-password-123'
    const keyPair = await generateKeyPair(password, 1000)

    expect(keyPair.privateKey).toBeInstanceOf(Uint8Array)
    expect(keyPair.publicKey).toBeInstanceOf(Uint8Array)
    expect(keyPair.authHash).toBeTruthy()

    expect(keyPair.privateKey.length).toBe(32)
    expect(keyPair.publicKey.length).toBe(65) // Uncompressed format
    expect(keyPair.authHash.length).toBe(64) // 32 bytes in hex
  })

  it('should generate same key pair for same password', async () => {
    const password = 'same-password'
    const keyPair1 = await generateKeyPair(password, 1000)
    const keyPair2 = await generateKeyPair(password, 1000)

    expect(keyPair1.privateKey).toEqual(keyPair2.privateKey)
    expect(keyPair1.publicKey).toEqual(keyPair2.publicKey)
    expect(keyPair1.authHash).toBe(keyPair2.authHash)
  })

  it('should generate different key pairs for different passwords', async () => {
    const keyPair1 = await generateKeyPair('password1', 1000)
    const keyPair2 = await generateKeyPair('password2', 1000)

    expect(keyPair1.privateKey).not.toEqual(keyPair2.privateKey)
    expect(keyPair1.publicKey).not.toEqual(keyPair2.publicKey)
    expect(keyPair1.authHash).not.toBe(keyPair2.authHash)
  })

  it('should generate different authHash from private key', async () => {
    const password = 'test-password'
    const keyPair = await generateKeyPair(password, 1000)

    // authHash should be different from private key hex
    const privateKeyHex = Array.from(keyPair.privateKey)
      .map(b => b.toString(16).padStart(2, '0'))
      .join('')

    expect(keyPair.authHash).not.toBe(privateKeyHex)
  })
})

describe('Key Storage and Restoration', () => {
  beforeEach(async () => {
    await clearKeys()
  })

  afterEach(async () => {
    await clearKeys()
  })

  it('should save and restore private key (session storage)', async () => {
    const keyPair = await generateKeyPair('test-password', 1000)

    await savePrivateKey(keyPair.privateKey, false)

    expect(hasStoredKey()).toBe(true)

    const restored = await restorePrivateKey()

    expect(restored).toEqual(keyPair.privateKey)
  })

  it('should save and restore private key (persistent storage)', async () => {
    const keyPair = await generateKeyPair('test-password', 1000)

    await savePrivateKey(keyPair.privateKey, true)

    expect(hasStoredKey()).toBe(true)

    const restored = await restorePrivateKey()

    expect(restored).toEqual(keyPair.privateKey)
  })

  it('should return null when no key is stored', async () => {
    const restored = await restorePrivateKey()

    expect(restored).toBeNull()
    expect(hasStoredKey()).toBe(false)
  })

  it('should clear stored keys', async () => {
    const keyPair = await generateKeyPair('test-password', 1000)

    await savePrivateKey(keyPair.privateKey, false)
    expect(hasStoredKey()).toBe(true)

    await clearKeys()

    expect(hasStoredKey()).toBe(false)

    const restored = await restorePrivateKey()
    expect(restored).toBeNull()
  })

  it('should overwrite existing key', async () => {
    const keyPair1 = await generateKeyPair('password1', 1000)
    const keyPair2 = await generateKeyPair('password2', 1000)

    await savePrivateKey(keyPair1.privateKey, false)
    await savePrivateKey(keyPair2.privateKey, false)

    const restored = await restorePrivateKey()

    expect(restored).toEqual(keyPair2.privateKey)
    expect(restored).not.toEqual(keyPair1.privateKey)
  })
})

describe('Public Key Derivation', () => {
  it('should derive public key from private key', async () => {
    const keyPair = await generateKeyPair('test-password', 1000)

    const derivedPublicKey = getPublicKeyFromPrivate(keyPair.privateKey)

    expect(derivedPublicKey).toEqual(keyPair.publicKey)
  })

  it('should derive same public key multiple times', async () => {
    const keyPair = await generateKeyPair('test-password', 1000)

    const publicKey1 = getPublicKeyFromPrivate(keyPair.privateKey)
    const publicKey2 = getPublicKeyFromPrivate(keyPair.privateKey)

    expect(publicKey1).toEqual(publicKey2)
  })
})
