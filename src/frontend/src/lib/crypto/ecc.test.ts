/**
 * Tests for ECC (Elliptic Curve Cryptography) module
 */

import { describe, it, expect } from 'vitest'
import {
  derivePrivateKey,
  derivePublicKey,
  computeSharedSecret,
  isValidPublicKey,
  toHex,
  fromHex,
  toBase64,
  fromBase64,
} from '../ecc'

describe('Key Derivation', () => {
  it('should derive private key from password', () => {
    const password = 'test-password-123'
    const privateKey = derivePrivateKey(password, 1000) // Use fewer iterations for tests

    expect(privateKey).toBeInstanceOf(Uint8Array)
    expect(privateKey.length).toBe(32)
  })

  it('should derive same private key for same password', () => {
    const password = 'same-password'
    const key1 = derivePrivateKey(password, 1000)
    const key2 = derivePrivateKey(password, 1000)

    expect(key1).toEqual(key2)
  })

  it('should derive different keys for different passwords', () => {
    const key1 = derivePrivateKey('password1', 1000)
    const key2 = derivePrivateKey('password2', 1000)

    expect(key1).not.toEqual(key2)
  })

  it('should derive public key from private key', () => {
    const privateKey = derivePrivateKey('test-password', 1000)
    const publicKey = derivePublicKey(privateKey)

    expect(publicKey).toBeInstanceOf(Uint8Array)
    expect(publicKey.length).toBe(65) // Uncompressed format
  })

  it('should derive compressed public key', () => {
    const privateKey = derivePrivateKey('test-password', 1000)
    const publicKey = derivePublicKey(privateKey, true)

    expect(publicKey).toBeInstanceOf(Uint8Array)
    expect(publicKey.length).toBe(33) // Compressed format
  })

  it('should derive same public key from same private key', () => {
    const privateKey = derivePrivateKey('test-password', 1000)
    const publicKey1 = derivePublicKey(privateKey)
    const publicKey2 = derivePublicKey(privateKey)

    expect(publicKey1).toEqual(publicKey2)
  })
})

describe('ECDH Shared Secret', () => {
  it('should compute same shared secret for both parties', () => {
    // Alice generates keys
    const alicePrivate = derivePrivateKey('alice-password', 1000)
    const alicePublic = derivePublicKey(alicePrivate)

    // Bob generates keys
    const bobPrivate = derivePrivateKey('bob-password', 1000)
    const bobPublic = derivePublicKey(bobPrivate)

    // Alice computes shared secret
    const aliceShared = computeSharedSecret(alicePrivate, bobPublic)

    // Bob computes shared secret
    const bobShared = computeSharedSecret(bobPrivate, alicePublic)

    // Both should get the same shared secret
    expect(aliceShared).toEqual(bobShared)
    expect(aliceShared.length).toBe(32)
  })

  it('should compute different shared secrets for different key pairs', () => {
    const alice1Private = derivePrivateKey('alice1', 1000)

    const alice2Private = derivePrivateKey('alice2', 1000)

    const bobPrivate = derivePrivateKey('bob', 1000)
    const bobPublic = derivePublicKey(bobPrivate)

    const shared1 = computeSharedSecret(alice1Private, bobPublic)
    const shared2 = computeSharedSecret(alice2Private, bobPublic)

    expect(shared1).not.toEqual(shared2)
  })
})

describe('Public Key Validation', () => {
  it('should validate correct public key', () => {
    const privateKey = derivePrivateKey('test', 1000)
    const publicKey = derivePublicKey(privateKey)

    expect(isValidPublicKey(publicKey)).toBe(true)
  })

  it('should reject invalid public key', () => {
    const invalidKey = new Uint8Array(65).fill(0)

    expect(isValidPublicKey(invalidKey)).toBe(false)
  })

  it('should reject random bytes as public key', () => {
    const randomBytes = crypto.getRandomValues(new Uint8Array(65))

    // Most random bytes won't be valid points on the curve
    // This test might occasionally pass if random bytes happen to be valid
    expect(isValidPublicKey(randomBytes)).toBe(false)
  })
})

describe('Hex Conversion', () => {
  it('should convert bytes to hex and back', () => {
    const original = new Uint8Array([0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef])
    const hex = toHex(original)
    const converted = fromHex(hex)

    expect(hex).toBe('0123456789abcdef')
    expect(converted).toEqual(original)
  })

  it('should handle empty array', () => {
    const empty = new Uint8Array([])
    const hex = toHex(empty)
    const converted = fromHex(hex)

    expect(hex).toBe('')
    expect(converted).toEqual(empty)
  })

  it('should pad single digit hex values', () => {
    const bytes = new Uint8Array([0x00, 0x01, 0x0f])
    const hex = toHex(bytes)

    expect(hex).toBe('00010f')
  })
})

describe('Base64 Conversion', () => {
  it('should convert bytes to base64 and back', () => {
    const original = new Uint8Array([72, 101, 108, 108, 111]) // "Hello"
    const base64 = toBase64(original)
    const converted = fromBase64(base64)

    expect(base64).toBe('SGVsbG8=')
    expect(converted).toEqual(original)
  })

  it('should handle empty array', () => {
    const empty = new Uint8Array([])
    const base64 = toBase64(empty)
    const converted = fromBase64(base64)

    expect(base64).toBe('')
    expect(converted).toEqual(empty)
  })

  it('should handle binary data', () => {
    const binary = crypto.getRandomValues(new Uint8Array(32))
    const base64 = toBase64(binary)
    const converted = fromBase64(base64)

    expect(converted).toEqual(binary)
  })
})
