import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { changePassword, logout } from '@/lib/api';
import { generateKeyPair, restorePrivateKey, savePrivateKey, clearKeys } from '@/lib/crypto';
import { getChats } from '@/lib/api/messages';
import { decryptChatKey, encryptChatKey } from '@/lib/crypto/encryption';
import type { ChatKeyUpdate } from '@/types/api';
import './ChangePasswordPage.css';

export function ChangePasswordPage() {
  const navigate = useNavigate();
  const [oldPassword, setOldPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [step, setStep] = useState<'input' | 'processing'>('input');
  const [userLogin, setUserLogin] = useState<string>('');

  useEffect(() => {
    const login = localStorage.getItem('user_login');
    if (login) {
      setUserLogin(login);
    }
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (!oldPassword || !newPassword) {
      setError('All fields are required');
      return;
    }

    if (newPassword.length < 8) {
      setError('New password must be at least 8 characters');
      return;
    }

    if (newPassword !== confirmPassword) {
      setError('New passwords do not match');
      return;
    }

    if (oldPassword === newPassword) {
      setError('New password must be different from old password');
      return;
    }

    setLoading(true);
    setStep('processing');

    try {
      // Step 1: Restore old private key
      const oldPrivateKey = restorePrivateKey();
      if (!oldPrivateKey) {
        throw new Error('No private key found. Please login again.');
      }

      // Step 2: Generate old and new key pairs
      const oldKeyPair = await generateKeyPair(oldPassword);
      const newKeyPair = await generateKeyPair(newPassword);

      // Step 3: Get user info (we need login)
      // Assuming we store login in localStorage or get it from context
      const login = localStorage.getItem('user_login');
      if (!login) {
        throw new Error('User login not found. Please login again.');
      }

      // Step 4: Get all chats
      const chats = await getChats();
      const chatKeys: ChatKeyUpdate[] = [];

      // Step 5: Re-encrypt chat keys
      for (const chat of chats) {
        try {
          // Decrypt chat key with old password
          const chatKey = await decryptChatKey(
            chat.encrypted_key,
            oldPassword
          );

          // Encrypt chat key with new password
          const encryptedData = await encryptChatKey(
            chatKey,
            newPassword
          );

          chatKeys.push({
            companion_id: chat.companion_id,
            encrypted_key: encryptedData.ciphertext,
            key_iv: encryptedData.nonce,
          });
        } catch (err) {
          console.error(`Failed to re-encrypt key for chat ${chat.id}:`, err);
          // Continue with other chats
        }
      }

      // Step 6: Change password on server
      await changePassword({
        login,
        old_auth_hash: oldKeyPair.authHash,
        new_auth_hash: newKeyPair.authHash,
        new_public_key: Array.from(newKeyPair.publicKey)
          .map(b => b.toString(16).padStart(2, '0'))
          .join(''),
        chat_keys: chatKeys.length > 0 ? chatKeys : undefined,
      });

      // Step 7: Clear old keys and save new ones
      clearKeys();
      savePrivateKey(newKeyPair.privateKey, true); // Always save to localStorage after password change

      // Step 8: Redirect to login
      navigate('/login', {
        state: { message: 'Password changed successfully! Please login with your new password.' }
      });
    } catch (err) {
      console.error('Password change error:', err);
      setError(err instanceof Error ? err.message : 'Failed to change password');
      setStep('input');
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = async () => {
    try {
      await logout();
      clearKeys();
      navigate('/login');
    } catch (error) {
      console.error('Logout error:', error);
      clearKeys();
      navigate('/login');
    }
  };

  return (
    <div className="change-password-page">
      <div className="change-password-container">
        <div className="settings-header">
          <h1>Settings</h1>
          <button
            onClick={() => navigate('/chats')}
            className="btn-back"
            title="Back to chats"
          >
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="m15 18-6-6 6-6"/>
            </svg>
          </button>
        </div>

        <div className="settings-section profile-section">
          <div className="profile-info">
            <div className="profile-avatar">
              {userLogin ? userLogin.substring(0, 2).toUpperCase() : 'U'}
            </div>
            <div className="profile-details">
              <h2>{userLogin || 'User'}</h2>
              <p className="profile-subtitle">Your account</p>
            </div>
          </div>
        </div>

        <div className="settings-section">
          <h2>Change Password</h2>
          <p className="subtitle">Update your password and re-encrypt your chats</p>

        {step === 'processing' && (
          <div className="processing-info">
            <p>🔄 Processing password change...</p>
            <p className="help-text">This may take a moment as we re-encrypt your chat keys</p>
          </div>
        )}

        <form onSubmit={handleSubmit} className="change-password-form">
          {error && (
            <div className="error-message">
              {error}
            </div>
          )}

          <div className="form-group">
            <label htmlFor="oldPassword">Current Password</label>
            <input
              id="oldPassword"
              type="password"
              value={oldPassword}
              onChange={(e) => setOldPassword(e.target.value)}
              placeholder="Enter your current password"
              disabled={loading}
              autoComplete="current-password"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="newPassword">New Password</label>
            <input
              id="newPassword"
              type="password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              placeholder="At least 8 characters"
              disabled={loading}
              autoComplete="new-password"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="confirmPassword">Confirm New Password</label>
            <input
              id="confirmPassword"
              type="password"
              value={confirmPassword}
              onChange={(e) => setConfirmPassword(e.target.value)}
              placeholder="Re-enter your new password"
              disabled={loading}
              autoComplete="new-password"
              required
            />
          </div>

          <button 
            type="submit" 
            className="btn-primary"
            disabled={loading}
          >
            {loading ? 'Changing Password...' : 'Change Password'}
          </button>
        </form>

        <div className="form-footer">
          <p>
            <a href="/chats" onClick={(e) => {
              e.preventDefault();
              navigate('/chats');
            }}>
              Cancel and go back
            </a>
          </p>
        </div>
        </div>

        <div className="settings-section danger-zone">
          <h2>Danger Zone</h2>
          <p className="subtitle">Logout from your account</p>
          <button
            onClick={handleLogout}
            className="btn-logout"
            type="button"
          >
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
              <polyline points="16 17 21 12 16 7"></polyline>
              <line x1="21" y1="12" x2="9" y2="12"></line>
            </svg>
            Logout
          </button>
        </div>
      </div>
    </div>
  );
}
