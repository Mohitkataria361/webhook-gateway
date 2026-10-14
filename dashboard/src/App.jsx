import React, { useState, useEffect } from 'react';
import LoginView from './components/LoginView';
import EndpointManager from './components/EndpointManager';
import LogStream from './components/LogStream';
import { getMe, sendWebhook } from './api/webhookApi';

export default function App() {
  const [token, setToken] = useState(() => localStorage.getItem('token') || localStorage.getItem('api_key') || '');
  const [user, setUser] = useState(null);
  const [loadingUser, setLoadingUser] = useState(true);
  const [authError, setAuthError] = useState(null);

  // --- Send Webhook Form State ---
  const [sendForm, setSendForm] = useState({
    endpointId: '',
    eventType: 'order.created',
    payload: '{\n  "order_id": "ORD-9842",\n  "amount": 149.50,\n  "currency": "USD"\n}',
    idempotencyKey: crypto.randomUUID(),
  });
  const [sending, setSending] = useState(false);
  const [sendResult, setSendResult] = useState(null);
  const [sendError, setSendError] = useState(null);

  // Check URL parameters for OAuth token or error on callback redirect
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const urlToken = params.get('token');
    const urlErr = params.get('auth_error');

    if (urlToken) {
      localStorage.setItem('token', urlToken);
      localStorage.setItem('api_key', urlToken);
      setToken(urlToken);
      // Clean up URL without triggering a page reload
      window.history.replaceState({}, document.title, window.location.pathname);
    } else if (urlErr) {
      setAuthError(decodeURIComponent(urlErr));
      window.history.replaceState({}, document.title, window.location.pathname);
    }
  }, []);

  // Fetch authenticated user profile
  useEffect(() => {
    if (!token) {
      setUser(null);
      setLoadingUser(false);
      return;
    }

    setLoadingUser(true);
    getMe()
      .then(({ data }) => {
        setUser(data?.user || null);
      })
      .catch((err) => {
        console.warn('Failed to load user profile with current token:', err);
        localStorage.removeItem('token');
        localStorage.removeItem('api_key');
        setToken('');
        setUser(null);
      })
      .finally(() => {
        setLoadingUser(false);
      });

    const handleUnauthorized = () => {
      setToken('');
      setUser(null);
      localStorage.removeItem('token');
      localStorage.removeItem('api_key');
    };
    window.addEventListener('auth:unauthorized', handleUnauthorized);
    return () => window.removeEventListener('auth:unauthorized', handleUnauthorized);
  }, [token]);

  const handleLogout = () => {
    localStorage.removeItem('token');
    localStorage.removeItem('api_key');
    setToken('');
    setUser(null);
    setSendResult(null);
    setSendError(null);
  };

  const handleSend = async (e) => {
    e.preventDefault();
    setSending(true);
    setSendError(null);
    setSendResult(null);

    try {
      let parsedPayload;
      try {
        parsedPayload = JSON.parse(sendForm.payload);
      } catch {
        throw new Error('Payload must be valid JSON');
      }

      const { data } = await sendWebhook({ ...sendForm, payload: parsedPayload });
      setSendResult(data);
      // Rotate idempotency key
      setSendForm((f) => ({ ...f, idempotencyKey: crypto.randomUUID() }));
    } catch (err) {
      setSendError(err.response?.data?.error || err.message);
    } finally {
      setSending(false);
    }
  };

  // ── 1. Loading State ──
  if (loadingUser) {
    return (
      <div className="loading-screen">
        <div className="spinner"></div>
        <p>Verifying secure session…</p>
      </div>
    );
  }

  // ── 2. Unauthenticated State (Login / Sign Up Gate) ──
  if (!token || !user) {
    return (
      <LoginView
        authError={authError}
        onLoginSuccess={(userData, userToken) => {
          setToken(userToken);
          setUser(userData);
          setAuthError(null);
        }}
      />
    );
  }

  // ── 3. Authenticated Dashboard ──
  const userAvatar =
    user.avatar_url ||
    `https://api.dicebear.com/7.x/identicon/svg?seed=${encodeURIComponent(user.email || user.name)}`;

  return (
    <div className="app">
      {/* ── Top Header ── */}
      <header className="app-header">
        <div className="header-inner">
          <div className="logo-group">
            <div className="logo">
              <span className="logo-icon">⚡</span>
              <span className="logo-text">Webhook Gateway</span>
            </div>
            <span className="tenant-badge">Isolated Tenant</span>
          </div>

          <div className="user-profile-bar">
            <div className="user-info-card">
              <img src={userAvatar} alt={user.name} className="user-avatar" />
              <div className="user-meta">
                <div className="user-name-row">
                  <strong className="user-name">{user.name}</strong>
                  <span className={`provider-pill provider-${user.provider}`}>
                    {user.provider}
                  </span>
                </div>
                <span className="user-email">{user.email}</span>
              </div>
            </div>

            <button
              id="logout-btn"
              className="btn-ghost btn-sm logout-btn"
              onClick={handleLogout}
              title="Sign out of this session"
            >
              Sign Out
            </button>
          </div>
        </div>
      </header>

      {/* ── Main Workspace ── */}
      <main className="app-main">
        {/* ── Top Grid: Endpoints & Dispatcher ── */}
        <div className="panels-grid">
          {/* User's Isolated Webhook Endpoints / Hooks */}
          <EndpointManager
            selectedEndpointId={sendForm.endpointId}
            onSelectEndpoint={(id) => setSendForm((f) => ({ ...f, endpointId: id }))}
          />

          {/* Send Webhook Event */}
          <div className="panel send-panel">
            <div className="panel-header-row">
              <div>
                <h3 className="panel-title">Dispatch Webhook Event</h3>
                <p className="panel-subtitle">Enqueue an event to be cryptographically signed and dispatched</p>
              </div>
            </div>

            <form className="form" onSubmit={handleSend}>
              <label className="field">
                <span>Target Endpoint ID</span>
                <input
                  id="send-endpoint-id"
                  type="text"
                  placeholder="Select an endpoint on the left or paste UUID"
                  value={sendForm.endpointId}
                  onChange={(e) => setSendForm({ ...sendForm, endpointId: e.target.value })}
                  required
                />
              </label>

              <label className="field">
                <div className="field-header">
                  <span>Event Type</span>
                  <div className="preset-chips">
                    {['order.created', 'payment.success', 'customer.updated'].map((evt) => (
                      <button
                        key={evt}
                        type="button"
                        className="preset-chip"
                        onClick={() => setSendForm((f) => ({ ...f, eventType: evt }))}
                      >
                        {evt}
                      </button>
                    ))}
                  </div>
                </div>
                <input
                  id="send-event-type"
                  type="text"
                  placeholder="order.created"
                  value={sendForm.eventType}
                  onChange={(e) => setSendForm({ ...sendForm, eventType: e.target.value })}
                  required
                />
              </label>

              <label className="field">
                <div className="field-header">
                  <span>Payload (JSON)</span>
                  <button
                    type="button"
                    className="btn-link-xs"
                    onClick={() =>
                      setSendForm((f) => ({
                        ...f,
                        payload: JSON.stringify(
                          {
                            order_id: `ORD-${Math.floor(1000 + Math.random() * 9000)}`,
                            amount: 99.99,
                            currency: 'USD',
                            time: new Date().toISOString(),
                          },
                          null,
                          2
                        ),
                      }))
                    }
                  >
                    Generate Sample
                  </button>
                </div>
                <textarea
                  id="send-payload"
                  rows={4}
                  value={sendForm.payload}
                  onChange={(e) => setSendForm({ ...sendForm, payload: e.target.value })}
                  required
                />
              </label>

              <label className="field">
                <span>Idempotency Key (Deduplication)</span>
                <div className="idem-row">
                  <input
                    id="send-idempotency-key"
                    type="text"
                    value={sendForm.idempotencyKey}
                    onChange={(e) => setSendForm({ ...sendForm, idempotencyKey: e.target.value })}
                    required
                  />
                  <button
                    type="button"
                    className="btn-ghost"
                    onClick={() => setSendForm((f) => ({ ...f, idempotencyKey: crypto.randomUUID() }))}
                    title="Generate new idempotency key"
                  >
                    ↺
                  </button>
                </div>
              </label>

              <button
                id="send-btn"
                type="submit"
                className="btn-primary"
                disabled={sending}
              >
                {sending ? 'Dispatching…' : '⚡ Dispatch Webhook'}
              </button>
            </form>

            {sendResult && (
              <div className="result-box result-success animate-fade">
                ✓ Event Accepted & Queued: <strong>{sendResult.event_id}</strong>
                <p className="text-xs text-muted mt-1">
                  Status: PENDING (enqueued into RabbitMQ). Check the <strong>Live Delivery Log</strong> table below to watch the worker dispatch with HMAC signature in real-time.
                </p>
              </div>
            )}
            {sendError && <div className="result-box result-error">✗ {sendError}</div>}
          </div>
        </div>

        {/* ── User-Isolated Live Log Stream ── */}
        <LogStream token={token} user={user} />
      </main>
    </div>
  );
}
