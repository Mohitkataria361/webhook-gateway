import React, { useState, useEffect } from 'react';
import LogStream from './components/LogStream';
import { registerEndpoint, sendWebhook, adminGenerateKey, getEndpoints } from './api/webhookApi';

export default function App() {
  // --- API Key State ---
  const [apiKey, setApiKey] = useState(() => {
    return localStorage.getItem('api_key') || localStorage.getItem('token') || '';
  });
  const [copiedKey, setCopiedKey] = useState(false);
  const [generatingKey, setGeneratingKey] = useState(false);
  const [keyNotice, setKeyNotice] = useState(null);

  // --- Endpoints State ---
  const [endpoints, setEndpoints] = useState([]);
  const [loadingEndpoints, setLoadingEndpoints] = useState(false);

  // --- Register Endpoint State ---
  const [regForm, setRegForm] = useState({ targetUrl: '' });
  const [regResult, setRegResult] = useState(null);
  const [regError, setRegError] = useState(null);

  // --- Send Webhook State ---
  const [sendForm, setSendForm] = useState({
    endpointId: '',
    eventType: 'order.created',
    payload: '{"order_id": "12345", "amount": 99.99, "currency": "USD"}',
    idempotencyKey: crypto.randomUUID(),
  });
  const [sendResult, setSendResult] = useState(null);
  const [sendError, setSendError] = useState(null);

  // Save API key when changed
  const updateApiKey = (val) => {
    const trimmed = val.trim();
    setApiKey(trimmed);
    if (trimmed) {
      localStorage.setItem('api_key', trimmed);
      localStorage.setItem('token', trimmed);
    } else {
      localStorage.removeItem('api_key');
      localStorage.removeItem('token');
    }
  };

  // Fetch endpoints for current API key
  const loadEndpoints = async () => {
    if (!apiKey) {
      setEndpoints([]);
      return;
    }
    setLoadingEndpoints(true);
    try {
      const { data } = await getEndpoints();
      setEndpoints(data?.endpoints || []);
    } catch (err) {
      console.warn('Could not load endpoints:', err);
    } finally {
      setLoadingEndpoints(false);
    }
  };

  useEffect(() => {
    if (apiKey) {
      loadEndpoints();
    }
  }, [apiKey]);

  // --- Generate API Key Handler ---
  const handleGenerateKey = async () => {
    setGeneratingKey(true);
    setKeyNotice(null);
    try {
      const { data } = await adminGenerateKey();
      if (data?.api_key) {
        updateApiKey(data.api_key);
        setKeyNotice({
          key: data.api_key,
          userId: data.user_id,
          message: data.message || 'New API Key generated successfully.',
        });
      }
    } catch (err) {
      alert('Failed to generate API key: ' + (err.response?.data?.error || err.message));
    } finally {
      setGeneratingKey(false);
    }
  };

  const copyToClipboard = (text) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(true);
    setTimeout(() => setCopiedKey(false), 2000);
  };

  // --- Register Handler ---
  const handleRegister = async (e) => {
    e.preventDefault();
    setRegError(null);
    setRegResult(null);

    if (!apiKey) {
      setRegError('Please provide or generate an API Key first.');
      return;
    }

    try {
      const { data } = await registerEndpoint(regForm.targetUrl);
      setRegResult(data);
      setSendForm((prev) => ({ ...prev, endpointId: data.id }));
      setRegForm({ targetUrl: '' });
      loadEndpoints();
    } catch (err) {
      setRegError(err.response?.data?.error || err.message);
    }
  };

  // --- Send Handler ---
  const handleSend = async (e) => {
    e.preventDefault();
    setSendError(null);
    setSendResult(null);

    if (!apiKey) {
      setSendError('Please provide or generate an API Key first.');
      return;
    }

    try {
      let parsedPayload;
      try {
        parsedPayload = JSON.parse(sendForm.payload);
      } catch {
        throw new Error('Payload must be valid JSON');
      }

      const { data } = await sendWebhook({ ...sendForm, payload: parsedPayload });
      setSendResult(data);
      setSendForm((f) => ({ ...f, idempotencyKey: crypto.randomUUID() }));
    } catch (err) {
      setSendError(err.response?.data?.error || err.message);
    }
  };

  return (
    <div className="app">
      {/* ── Header ── */}
      <header className="app-header">
        <div className="header-inner">
          <div className="logo">
            <span className="logo-icon">⚡</span>
            <span className="logo-text">Webhook Gateway</span>
          </div>
          <p className="header-sub">Inspection & Delivery Gateway • Multi-Tenant B2B</p>
        </div>
      </header>

      <main className="app-main">
        {/* ── Developer API Key Bar ── */}
        <section className="api-key-bar">
          <div className="api-key-left">
            <span className="api-key-label">🔑 API Key:</span>
            <div className="api-key-input-group">
              <input
                id="api-key-input"
                type="text"
                placeholder="sk_live_... (or click Generate Key)"
                value={apiKey}
                onChange={(e) => updateApiKey(e.target.value)}
              />
              {apiKey && (
                <button
                  type="button"
                  className="search-clear"
                  onClick={() => copyToClipboard(apiKey)}
                  title="Copy API Key"
                >
                  {copiedKey ? '✓' : '📋'}
                </button>
              )}
            </div>
            <span className={`key-badge ${apiKey ? 'active' : 'missing'}`}>
              {apiKey ? 'Active' : 'No Key'}
            </span>
          </div>

          <div className="api-key-right">
            <button
              id="generate-key-btn"
              type="button"
              className="btn-generate"
              onClick={handleGenerateKey}
              disabled={generatingKey}
            >
              {generatingKey ? 'Generating…' : '✨ Generate Key'}
            </button>
          </div>
        </section>

        {keyNotice && (
          <div className="result-box result-success animate-fade" style={{ marginBottom: '24px' }}>
            <p><strong>✓ New Secret API Key Issued:</strong></p>
            <code className="secret-code">{keyNotice.key}</code>
            <p className="secret-warning">
              ⚠ {keyNotice.message} Only its SHA-256 hash is stored on the server.
            </p>
            <button
              type="button"
              className="btn-ghost"
              style={{ marginTop: '8px' }}
              onClick={() => setKeyNotice(null)}
            >
              Dismiss
            </button>
          </div>
        )}

        {/* ── Control Panels ── */}
        <div className="panels-grid">
          {/* Register Endpoint */}
          <div className="panel">
            <h3 className="panel-title">Register Webhook Endpoint</h3>
            <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Endpoints are automatically isolated and linked to your API key.
            </p>

            <form className="form" onSubmit={handleRegister}>
              <label className="field">
                <span>Target URL (HTTPS / HTTP)</span>
                <input
                  id="reg-target-url"
                  type="url"
                  placeholder="https://webhook.site/demo or http://localhost:9090/ok"
                  value={regForm.targetUrl}
                  onChange={(e) => setRegForm({ targetUrl: e.target.value })}
                  required
                />
              </label>
              <button id="register-btn" type="submit" className="btn-primary">
                Register Endpoint
              </button>
            </form>

            {regResult && (
              <div className="result-box result-success animate-fade">
                <p><strong>Endpoint ID:</strong> {regResult.id}</p>
                <p><strong>Signing Secret:</strong></p>
                <code className="secret-code">{regResult.secret_key}</code>
                <p className="secret-warning">
                  ⚠ Used by receivers to verify HMAC-SHA256 signatures in the <code>X-Hub-Signature-256</code> header.
                </p>
              </div>
            )}
            {regError && <div className="result-box result-error">✗ {regError}</div>}

            {/* List of Registered Endpoints for this Key */}
            {endpoints.length > 0 && (
              <div style={{ marginTop: '20px' }}>
                <span className="api-key-label" style={{ marginBottom: '8px', display: 'block' }}>
                  Your Registered Endpoints ({endpoints.length}):
                </span>
                <div className="endpoints-quicklist">
                  {endpoints.map((ep) => (
                    <div key={ep.id} className="endpoint-pill">
                      <span className="endpoint-pill-url" title={ep.target_url}>
                        {ep.target_url}
                      </span>
                      <button
                        type="button"
                        onClick={() => setSendForm((f) => ({ ...f, endpointId: ep.id }))}
                      >
                        Use in Form →
                      </button>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* Send Webhook */}
          <div className="panel">
            <h3 className="panel-title">Send Webhook Event</h3>
            <p style={{ fontSize: '0.82rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
              Publishes an asynchronous delivery job into RabbitMQ with exponential retries.
            </p>

            <form className="form" onSubmit={handleSend}>
              <label className="field">
                <span>Endpoint ID (UUID)</span>
                <input
                  id="send-endpoint-id"
                  type="text"
                  placeholder="Select above or paste Endpoint ID"
                  value={sendForm.endpointId}
                  onChange={(e) => setSendForm({ ...sendForm, endpointId: e.target.value })}
                  required
                />
              </label>
              <label className="field">
                <span>Event Type</span>
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
                <span>Payload (JSON)</span>
                <textarea
                  id="send-payload"
                  rows={4}
                  value={sendForm.payload}
                  onChange={(e) => setSendForm({ ...sendForm, payload: e.target.value })}
                  required
                />
              </label>
              <label className="field">
                <span>Idempotency Key</span>
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
              <button id="send-btn" type="submit" className="btn-primary">
                Send Webhook
              </button>
            </form>

            {sendResult && (
              <div className="result-box result-success animate-fade">
                ✓ Event queued: <strong>{sendResult.event_id}</strong>
              </div>
            )}
            {sendError && <div className="result-box result-error">✗ {sendError}</div>}
          </div>
        </div>

        {/* ── Live Log Stream ── */}
        <LogStream token={apiKey} />
      </main>
    </div>
  );
}
