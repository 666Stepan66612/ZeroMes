/**
 * Check if Web Crypto API is available
 * Required for encryption/decryption operations
 */
export function isCryptoAvailable(): boolean {
  return typeof window !== 'undefined' && 
         typeof window.crypto !== 'undefined' && 
         typeof window.crypto.subtle !== 'undefined';
}

/**
 * Get crypto.subtle with error handling
 * Throws descriptive error if not available
 */
export function getCryptoSubtle(): SubtleCrypto {
  if (!isCryptoAvailable()) {
    throw new Error(
      'Web Crypto API is not available. ' +
      'This application requires HTTPS or localhost to work. ' +
      'Please access the site via HTTPS or configure your server to use SSL/TLS.'
    );
  }
  return window.crypto.subtle;
}

/**
 * Check if running in secure context
 */
export function isSecureContext(): boolean {
  return typeof window !== 'undefined' && window.isSecureContext === true;
}
