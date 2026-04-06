/**
 * Authentication API client
 */

import { api } from './client'
import type {
  RegisterRequest,
  LoginRequest,
  ChangePasswordRequest,
  AuthResponse,
  RefreshTokenResponse,
  User,
} from '../../types/api'

/**
 * Register new user
 * 
 * @param data - Registration data (login, auth_hash, public_key)
 * @returns User data and JWT tokens
 */
export async function register(data: RegisterRequest): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/register', data)
}

/**
 * Login user
 * 
 * @param data - Login credentials (login, auth_hash)
 * @returns User data and JWT tokens
 */
export async function login(data: LoginRequest): Promise<AuthResponse> {
  return api.post<AuthResponse>('/auth/login', data)
}

/**
 * Refresh access token
 * 
 * @returns New token pair
 */
export async function refreshToken(): Promise<RefreshTokenResponse> {
  return api.post<RefreshTokenResponse>('/auth/refresh')
}

/**
 * Logout user
 * Invalidates refresh and access tokens
 */
export async function logout(): Promise<void> {
  await api.post<void>('/auth/logout')
}

/**
 * Search users by login
 * 
 * @param query - Search query (login substring)
 * @returns Array of matching users
 */
export async function searchUsers(query: string): Promise<User[]> {
  const response = await api.get<{ users: User[] }>('/auth/search', {
    params: { login: query },
  })
  return response.users
}

/**
 * Change user password
 * Also updates public key since it's derived from password
 * 
 * @param data - Password change data
 * @returns Success status
 */
export async function changePassword(
  data: ChangePasswordRequest
): Promise<{ success: boolean }> {
  return api.post<{ success: boolean }>('/auth/change-password', data)
}

/**
 * Get user's public key by user ID
 * Used for ECDH key exchange
 * 
 * @param userId - User ID
 * @returns User's public key
 */
export async function getUserPublicKey(userId: string): Promise<string> {
  // Search by ID using the new id parameter
  const response = await api.get<{ users: User[] }>('/auth/search', {
    params: { id: userId },
  })
  
  const users = response.users
  if (!users || users.length === 0) {
    console.error(`[getUserPublicKey] User ${userId} not found`)
    throw new Error(`User ${userId} not found`)
  }
  
  return users[0].public_key
}
