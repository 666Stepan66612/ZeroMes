/**
 * Эллиптическая криптография на кривой secp256k1
 * Используется для ECDH (Elliptic Curve Diffie-Hellman)
 */

import * as secp256k1 from '@noble/secp256k1'
import { sha256 } from '@noble/hashes/sha2.js'

/**
 * Генерация приватного ключа из пароля
 * Использует множественные итерации SHA256 для усиления
 * 
 * @param password - Пароль пользователя
 * @param iterations - Количество итераций хеширования (по умолчанию 10000)
 * @returns Приватный ключ (32 байта)
 */
export function derivePrivateKey(password: string, iterations = 10000): Uint8Array {
  let hash: Uint8Array = new TextEncoder().encode(password)
  
  // Множественные итерации SHA256 для защиты от brute-force
  for (let i = 0; i < iterations; i++) {
    hash = new Uint8Array(sha256(hash))
  }
  
  return hash  // 32 байта
}

/**
 * Генерация публичного ключа из приватного
 * Использует умножение на базовую точку G кривой secp256k1
 * 
 * @param privateKey - Приватный ключ (32 байта)
 * @param compressed - Использовать сжатый формат (33 байта вместо 65)
 * @returns Публичный ключ
 */
export function derivePublicKey(privateKey: Uint8Array, compressed = false): Uint8Array {
  return secp256k1.getPublicKey(privateKey, compressed)
}

/**
 * ECDH: Вычисление общего секрета между двумя сторонами
 * Алиса: sharedSecret = alicePrivateKey * bobPublicKey
 * Боб: sharedSecret = bobPrivateKey * alicePublicKey
 * Результат одинаковый благодаря свойствам эллиптических кривых
 * 
 * @param privateKey - Мой приватный ключ
 * @param publicKey - Публичный ключ собеседника
 * @returns Общий секрет (32 байта) для использования в AES
 */
export function computeSharedSecret(
  privateKey: Uint8Array,
  publicKey: Uint8Array
): Uint8Array {
  // Получить общую точку на кривой
  const sharedPoint = secp256k1.getSharedSecret(privateKey, publicKey)
  
  // Взять x координату (пропустить первый байт - префикс)
  // sharedPoint формат: [prefix, x(32 bytes), y(32 bytes)]
  const x = sharedPoint.slice(1, 33)
  
  // Хешировать x координату для получения ключа шифрования
  return new Uint8Array(sha256(x))  // 32 байта для AES-256
}

/**
 * Проверка валидности публичного ключа
 * 
 * @param publicKey - Публичный ключ для проверки
 * @returns true если ключ валиден
 */
export function isValidPublicKey(publicKey: Uint8Array): boolean {
  try {
    // Конвертируем в hex для проверки
    const hex = toHex(publicKey)
    secp256k1.Point.fromHex(hex)
    return true
  } catch {
    return false
  }
}

/**
 * Конвертация Uint8Array в hex string
 */
export function toHex(bytes: Uint8Array): string {
  return Array.from(bytes)
    .map(b => b.toString(16).padStart(2, '0'))
    .join('')
}

/**
 * Конвертация hex string в Uint8Array
 */
export function fromHex(hex: string): Uint8Array {
  const bytes = new Uint8Array(hex.length / 2)
  for (let i = 0; i < hex.length; i += 2) {
    bytes[i / 2] = parseInt(hex.substr(i, 2), 16)
  }
  return bytes
}

/**
 * Конвертация Uint8Array в base64
 */
export function toBase64(bytes: Uint8Array): string {
  return btoa(String.fromCharCode(...bytes))
}

/**
 * Конвертация base64 в Uint8Array
 */
export function fromBase64(base64: string): Uint8Array {
  const binary = atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes
}
