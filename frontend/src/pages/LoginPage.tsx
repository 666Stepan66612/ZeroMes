import { useState, useEffect } from 'react';
import type { FormEvent } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { login } from '@/lib/api';
import { generateKeyPair, savePrivateKey, clearKeys } from '@/lib/crypto';

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [loginValue, setLoginValue] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [successMessage, setSuccessMessage] = useState('');

  useEffect(() => {
    // Show success message from registration
    if (location.state?.message) {
      setSuccessMessage(location.state.message);
      // Clear the message from location state
      window.history.replaceState({}, document.title);
    }
  }, [location]);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccessMessage('');

    if (!loginValue.trim() || !password) {
      setError('Please enter login and password');
      return;
    }

    setLoading(true);

    try {
      console.log('[LoginPage] Starting login, rememberMe:', rememberMe);
      
      // Clear ALL old user data first (important for switching users)
      await clearKeys();
      localStorage.clear(); // Clear all localStorage including user_login, tokens, etc.
      sessionStorage.clear(); // Clear all sessionStorage
      
      // Clear all cookies (JWT tokens are stored in cookies)
      document.cookie.split(";").forEach((c) => {
        document.cookie = c
          .replace(/^ +/, "")
          .replace(/=.*/, "=;expires=" + new Date().toUTCString() + ";path=/");
      });
      console.log('[LoginPage] Old user data and cookies cleared');
      
      // Generate key pair from password
      const keyPair = await generateKeyPair(password);
      console.log('[LoginPage] Key pair generated');

      // Login
      await login({
        login: loginValue.trim(),
        auth_hash: keyPair.authHash,
      });
      console.log('[LoginPage] Login successful');

      // Save private key (sessionStorage or IndexedDB based on rememberMe)
      console.log('[LoginPage] Saving private key, rememberMe:', rememberMe);
      await savePrivateKey(keyPair.privateKey, rememberMe);
      console.log('[LoginPage] Private key saved');
      
      // Save NEW user login to localStorage
      localStorage.setItem('user_login', loginValue.trim());

      // Small delay to ensure storage is written
      await new Promise(resolve => setTimeout(resolve, 100));
      console.log('[LoginPage] Redirecting to chats');

      // Redirect to chats
      navigate('/chats');
    } catch (err) {
      console.error('Login error:', err);
      setError(
        err instanceof Error 
          ? err.message 
          : 'Login failed. Please check your credentials.'
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-page">
      <div className="login-container">
        <h1>Welcome Back</h1>
        <p className="subtitle">Login to your secure messenger</p>

        <form onSubmit={handleSubmit} className="login-form">
          {successMessage && (
            <div className="success-message">
              {successMessage}
            </div>
          )}

          {error && (
            <div className="error-message">
              {error}
            </div>
          )}

          <div className="form-group">
            <label htmlFor="login">Login</label>
            <input
              id="login"
              type="text"
              value={loginValue}
              onChange={(e) => setLoginValue(e.target.value)}
              placeholder="Enter your login"
              disabled={loading}
              autoComplete="username"
              required
            />
          </div>

          <div className="form-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter your password"
              disabled={loading}
              autoComplete="current-password"
              required
            />
          </div>

          <div className="form-group checkbox">
            <label>
              <input
                type="checkbox"
                checked={rememberMe}
                onChange={(e) => setRememberMe(e.target.checked)}
                disabled={loading}
              />
              <span>Remember me</span>
            </label>
            <p className="help-text">
              Stay logged in even after closing browser (recommended)
            </p>
          </div>

          <button 
            type="submit" 
            className="btn-primary"
            disabled={loading}
          >
            {loading ? 'Logging in...' : 'Login'}
          </button>
        </form>

        <div className="form-footer">
          <p>
            Don't have an account?{' '}
            <a href="/register" onClick={(e) => {
              e.preventDefault();
              navigate('/register');
            }}>
              Register here
            </a>
          </p>
        </div>

        <div className="security-notice">
          <p>End-to-end encrypted messaging</p>
          <p>Your messages are secure and private</p>
        </div>
      </div>
    </div>
  );
}
