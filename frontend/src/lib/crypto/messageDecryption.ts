/**
 * Helper for message decryption
 * Eliminates code duplication
 */

import type { EncryptedMessage } from '@/types/crypto';

/**
 * Decrypt a single message using chat key
 * 
 * @param encryptedContent - JSON string with encrypted message
 * @param chatKey - Chat encryption key
 * @returns Decrypted message text
 */
export async function decryptMessageContent(
  encryptedContent: string,
  chatKey: Uint8Array
): Promise<string> {
  try {
    // Parse encrypted message
    const encryptedMsg: EncryptedMessage = JSON.parse(encryptedContent);
    
    // Import key for decryption
    const key = await crypto.subtle.importKey(
      'raw',
      chatKey as BufferSource,
      { name: 'AES-GCM' },
      false,
      ['decrypt']
    );
    
    // Decode base64
    const ciphertextBytes = Uint8Array.from(atob(encryptedMsg.ciphertext), c => c.charCodeAt(0));
    const nonceBytes = Uint8Array.from(atob(encryptedMsg.nonce), c => c.charCodeAt(0));
    
    // Decrypt
    const plaintext = await crypto.subtle.decrypt(
      { name: 'AES-GCM', iv: nonceBytes },
      key,
      ciphertextBytes
    );
    
    return new TextDecoder().decode(plaintext);
  } catch (error) {
    console.error('[decryptMessageContent] Decryption failed:', error);
    throw new Error('Failed to decrypt message');
  }
}
