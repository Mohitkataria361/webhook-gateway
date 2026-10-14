import React, { useState, useEffect } from 'react';
import { getAuthConfig, devLogin, getOAuthUrl } from '../api/webhookApi';

export default function LoginView({ onLoginSuccess, authError }) {
  const [authConfig, setAuthConfig] = useState({ github_enabled: false, google_enabled: false, dev_enabled: true });
  const [activeTab, setActiveTab] = useState('login'); // 'login' | 'signup'

  // Form fields
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [username, setUsername] = useState('');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(authError || null);

  useEffect(() => {
    getAuthConfig()
      .then(({ data }) => setAuthConfig(data))
      .catch((err) => {
        console.warn('Could not load auth config from server:', err);
      });
  }, []);

  const handleAuthSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const uName = (username || email.split('@')[0] || 'user').trim().toLowerCase();
      const uEmail = (email || `${uName}@example.com`).trim();
      const dName = (name || uName.charAt(0).toUpperCase() + uName.slice(1)).trim();

      const { data } = await devLogin(uName, uEmail, dName);
      localStorage.setItem('token', data.token);
      localStorage.setItem('api_key', data.token);
      onLoginSuccess(data.user, data.token);
    } catch (err) {
      setError(err.response?.data?.error || err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleQuickDemo = async (uName, uEmail, dName) => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await devLogin(uName, uEmail, dName);
      localStorage.setItem('token', data.token);
      localStorage.setItem('api_key', data.token);
      onLoginSuccess(data.user, data.token);
    } catch (err) {
      setError(err.response?.data?.error || err.message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-card animate-fade">
        {/* ── Brand Header ── */}
        <div className="login-header">
          <div className="login-logo">
            <span className="login-logo-icon">⚡</span>
            <span className="login-logo-text">Webhook Gateway</span>
          </div>
          <h2>{activeTab === 'login' ? 'Welcome Back' : 'Create Your Account'}</h2>
          <p className="login-subtitle">
            {activeTab === 'login'
              ? 'Sign in to access your isolated webhooks, secrets, and live delivery logs.'
              : 'Sign up to get your own isolated tenant with private HMAC hooks and streams.'}
          </p>
        </div>

        {/* ── Sign In / Sign Up Tabs ── */}
        <div className="auth-tabs">
          <button
            type="button"
            className={`auth-tab ${activeTab === 'login' ? 'active' : ''}`}
            onClick={() => { setActiveTab('login'); setError(null); }}
          >
            Sign In
          </button>
          <button
            type="button"
            className={`auth-tab ${activeTab === 'signup' ? 'active' : ''}`}
            onClick={() => { setActiveTab('signup'); setError(null); }}
          >
            Sign Up
          </button>
        </div>

        {error && (
          <div className="result-box result-error mb-4">
            ✗ {error}
          </div>
        )}

        {/* ── OAuth 1-Click Providers ── */}
        <div className="oauth-section">
          <a
            href={getOAuthUrl('github')}
            className="oauth-btn oauth-github"
            title={authConfig.github_enabled ? 'Authenticate with GitHub' : 'Configure GITHUB_CLIENT_ID and SECRET in .env'}
          >
            <svg className="oauth-icon" viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
            </svg>
            <span>{activeTab === 'login' ? 'Sign in with GitHub' : 'Sign up with GitHub'}</span>
            {!authConfig.github_enabled && <span className="oauth-chip">Setup in .env</span>}
          </a>

          <a
            href={getOAuthUrl('google')}
            className="oauth-btn oauth-google"
            title={authConfig.google_enabled ? 'Authenticate with Google' : 'Configure GOOGLE_CLIENT_ID and SECRET in .env'}
          >
            <svg className="oauth-icon" viewBox="0 0 24 24" width="20" height="20">
              <path fill="#4285F4" d="M23.745 12.27c0-.7-.06-1.4-.19-2.07H12v4.51h6.6c-.29 1.52-1.14 2.8-2.4 3.65v3h3.86c2.26-2.09 3.685-5.17 3.685-9.09z"/>
              <path fill="#34A853" d="M12 24c3.24 0 5.95-1.08 7.93-2.91l-3.86-3c-1.08.72-2.45 1.16-4.07 1.16-3.13 0-5.78-2.11-6.73-4.96H1.26v3.09C3.26 21.3 7.36 24 12 24z"/>
              <path fill="#FBBC05" d="M5.27 14.29c-.25-.72-.38-1.49-.38-2.29s.13-1.57.38-2.29V6.62H1.26C.46 8.23 0 10.06 0 12s.46 3.77 1.26 5.38l4.01-3.09z"/>
              <path fill="#EA4335" d="M12 4.75c1.77 0 3.35.61 4.6 1.8l3.42-3.42C17.95 1.19 15.24 0 12 0 7.36 0 3.26 2.7 1.26 6.62l4.01 3.09c.95-2.85 3.6-4.96 6.73-4.96z"/>
            </svg>
            <span>{activeTab === 'login' ? 'Sign in with Google' : 'Sign up with Google'}</span>
            {!authConfig.google_enabled && <span className="oauth-chip">Setup in .env</span>}
          </a>
        </div>

        {/* ── Divider ── */}
        <div className="login-divider">
          <span>OR CONTINUE WITH EMAIL / USERNAME</span>
        </div>

        {/* ── Email / Username Form ── */}
        <form className="form" onSubmit={handleAuthSubmit}>
          {activeTab === 'signup' && (
            <label className="field">
              <span>Full Name</span>
              <input
                type="text"
                placeholder="e.g. Sarah Connor"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </label>
          )}

          <label className="field">
            <span>{activeTab === 'signup' ? 'Email Address' : 'Email or Username'}</span>
            <input
              type="text"
              placeholder={activeTab === 'signup' ? 'sarah@company.com' : 'username or email'}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </label>

          <button type="submit" className="btn-primary w-full" disabled={loading}>
            {loading ? 'Please wait…' : activeTab === 'login' ? 'Sign In' : 'Create Account'}
          </button>
        </form>

        {/* ── Quick Demo Accounts ── */}
        <div className="login-divider" style={{ marginTop: '20px' }}>
          <span>OR 1-CLICK DEMO ACCOUNTS</span>
        </div>

        <div className="demo-accounts-section">
          <p className="demo-hint">
            Instantly switch users to test multi-tenant hook and log isolation:
          </p>

          <div className="demo-pills-grid">
            <button
              type="button"
              className="demo-pill"
              onClick={() => handleQuickDemo('alice', 'alice@company.com', 'Alice (Finance)')}
              disabled={loading}
            >
              <img
                src="https://api.dicebear.com/7.x/identicon/svg?seed=alice"
                alt="Alice"
                className="demo-avatar"
              />
              <div className="demo-info">
                <strong>Alice</strong>
                <span>alice@company.com (Finance)</span>
              </div>
            </button>

            <button
              type="button"
              className="demo-pill"
              onClick={() => handleQuickDemo('bob', 'bob@company.com', 'Bob (Payments)')}
              disabled={loading}
            >
              <img
                src="https://api.dicebear.com/7.x/identicon/svg?seed=bob"
                alt="Bob"
                className="demo-avatar"
              />
              <div className="demo-info">
                <strong>Bob</strong>
                <span>bob@company.com (Payments)</span>
              </div>
            </button>
          </div>
        </div>

        {/* ── Security Note ── */}
        <div className="login-footer">
          <p>
            🔒 Every user account has completely isolated webhook endpoints, HMAC signing secrets, and live delivery logs.
          </p>
        </div>
      </div>
    </div>
  );
}
