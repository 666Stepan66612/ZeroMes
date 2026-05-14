/**
 * Complete logout utility
 * Clears all user data and disconnects WebSocket
 */

import { logout as logoutAPI } from '@/lib/api/auth';
import { resetWebSocketClient } from '@/lib/api/websocket';
import { clearKeys } from '@/lib/crypto';

/**
 * Perform complete logout
 * - Calls logout API to invalidate tokens on server
 * - Disconnects and resets WebSocket
 * - Clears all crypto keys (IndexedDB + localStorage + sessionStorage + memory)
 * - Clears all localStorage and sessionStorage
 */
export async function performLogout(): Promise<void> {
  console.log('[Logout] Starting complete logout...');

  // 1. Call logout API to invalidate tokens on server
  try {
    await logoutAPI();
    console.log('[Logout] Server logout successful');
  } catch (error) {
    console.error('[Logout] Server logout failed:', error);
    // Continue with cleanup even if API fails
  }

  // 2. Disconnect and reset WebSocket
  try {
    resetWebSocketClient();
    console.log('[Logout] WebSocket disconnected');
  } catch (error) {
    console.error('[Logout] WebSocket disconnect failed:', error);
  }

  // 3. Clear all crypto keys (IndexedDB + localStorage + sessionStorage + memory)
  try {
    await clearKeys();
    console.log('[Logout] Crypto keys cleared');
  } catch (error) {
    console.error('[Logout] Failed to clear crypto keys:', error);
  }

  // 4. Clear all localStorage data
  try {
    localStorage.clear();
    console.log('[Logout] localStorage cleared');
  } catch (error) {
    console.error('[Logout] Failed to clear localStorage:', error);
  }

  // 5. Clear all sessionStorage data
  try {
    sessionStorage.clear();
    console.log('[Logout] sessionStorage cleared');
  } catch (error) {
    console.error('[Logout] Failed to clear sessionStorage:', error);
  }

  console.log('[Logout] ✅ Complete logout finished');
}
