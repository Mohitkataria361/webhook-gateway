import React, { useState } from 'react';
import LogStream from './components/LogStream';
import { registerEndpoint, sendWebhook } from './api/webhookApi';

export default function App() {
  // --- Register Endpoint State ---
  const [regForm, setRegForm] = useState({ userId: '', targetUrl: '' });
  const [regResult, setRegResult] = useState(null);
  const [regError, setRegError] = useState(null);

  // --- Send Webhook State ---
  const [sendForm, setSendForm] = useState({
    endpointId: '',
    eventType: 'order.created',
    payload: '{"order_id": "12345", "amount": 99.99}',
    idempotencyKey: crypto.randomUUID(),
  });
  const [sendResult, setSendResult] = useState(null);
  const [sendError, setSendError] = useState(null);

  // --- Register Handler ---
  const handleRegister = async (e) => {
    e.preventDefault();
    setRegError(null);
    setRegResult(null);
    try {
      const { data } = await registerEndpoint(regForm.userId, regForm.targetUrl);
      setRegResult(data);
    } catch (err) {
      setRegError(err.response?.data?.error || err.message);
    }
  };

  // --- Send Handler ---
  const handleSend = async (e) => {
    e.preventDefault();
    setSendError(null);
    setSendResult(null);
    try {
      let parsedPayload;
      try { parsedPayload = JSON.parse(sendForm.payload); }
      catch { throw new Error('Payload must be valid JSON'); }

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
          <p className="header-sub">Inspection & Delivery Dashboard</p>
        </div>
      </header>

      <main className="app-main">
        {/* ── Control Panels ── */}
        <div className="panels-grid">

          {/* Register Endpoint */}
          <div className="panel">
            <h3 className="panel-title">Register Endpoint</h3>
            <form className="form" onSubmit={handleRegister}>
              <label className="field">
                <span>User ID (UUID)</span>
                <input
                  id="reg-user-id"
                  type="text"
                  placeholder="550e8400-e29b-41d4-a716-446655440000"
                  value={regForm.userId}
                  onChange={(e) => setRegForm({ ...regForm, userId: e.target.value })}
                  required
                />
              </label>
              <label className="field">
                <span>Target URL</span>
                <input
                  id="reg-target-url"
                  type="url"
                  placeholder="http://localhost:9090/ok"
                  value={regForm.targetUrl}
                  onChange={(e) => setRegForm({ ...regForm, targetUrl: e.target.value })}
                  required
                />
              </label>
              <button id="register-btn" type="submit" className="btn-primary">Register</button>
            </form>
            {regResult && (
              <div className="result-box result-success">
                <p><strong>Endpoint ID:</strong> {regResult.id}</p>
                <p><strong>Secret Key:</strong></p>
                <code className="secret-code">{regResult.secret_key}</code>
                <p className="secret-warning">⚠ Store this secret safely — it won't be shown again.</p>
              </div>
            )}
            {regError && <div className="result-box result-error">✗ {regError}</div>}
          </div>

          {/* Send Webhook */}
          <div className="panel">
            <h3 className="panel-title">Send Webhook Event</h3>
            <form className="form" onSubmit={handleSend}>
              <label className="field">
                <span>Endpoint ID (UUID)</span>
                <input
                  id="send-endpoint-id"
                  type="text"
                  placeholder="Endpoint UUID from registration"
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
                    title="Generate new key"
                  >↺</button>
                </div>
              </label>
              <button id="send-btn" type="submit" className="btn-primary">Send Webhook</button>
            </form>
            {sendResult && (
              <div className="result-box result-success">
                ✓ Event queued: <strong>{sendResult.event_id}</strong>
              </div>
            )}
            {sendError && <div className="result-box result-error">✗ {sendError}</div>}
          </div>
        </div>

        {/* ── Live Log Stream ── */}
        <LogStream />
      </main>
    </div>
  );
}
