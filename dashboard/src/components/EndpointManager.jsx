import React, { useState, useEffect } from 'react';
import { getEndpoints, registerEndpoint, deleteEndpoint } from '../api/webhookApi';

export default function EndpointManager({ onSelectEndpoint, selectedEndpointId }) {
  const [endpoints, setEndpoints] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  // Form state
  const [targetUrl, setTargetUrl] = useState('');
  const [registering, setRegistering] = useState(false);
  const [regResult, setRegResult] = useState(null);
  const [regError, setRegError] = useState(null);
  const [copiedId, setCopiedId] = useState(null);
  const [copiedSecret, setCopiedSecret] = useState(false);

  const fetchEndpoints = async () => {
    setLoading(true);
    setError(null);
    try {
      const { data } = await getEndpoints();
      setEndpoints(data?.endpoints || []);
    } catch (err) {
      setError(err.response?.data?.error || err.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchEndpoints();
  }, []);

  const handleRegister = async (e) => {
    e.preventDefault();
    setRegistering(true);
    setRegError(null);
    setRegResult(null);

    try {
      const { data } = await registerEndpoint(targetUrl);
      setRegResult(data);
      setTargetUrl('');
      fetchEndpoints();
      if (onSelectEndpoint) {
        onSelectEndpoint(data.id);
      }
    } catch (err) {
      setRegError(err.response?.data?.error || err.message);
    } finally {
      setRegistering(false);
    }
  };

  const handleDelete = async (id, e) => {
    e.stopPropagation();
    if (!window.confirm('Are you sure you want to delete this endpoint?')) return;
    try {
      await deleteEndpoint(id);
      setEndpoints((prev) => prev.filter((item) => item.id !== id));
      if (selectedEndpointId === id && onSelectEndpoint) {
        onSelectEndpoint('');
      }
    } catch (err) {
      alert(err.response?.data?.error || 'Failed to delete endpoint');
    }
  };

  const copyToClipboard = (text, type, id = null) => {
    navigator.clipboard.writeText(text);
    if (type === 'id') {
      setCopiedId(id);
      setTimeout(() => setCopiedId(null), 2000);
    } else if (type === 'secret') {
      setCopiedSecret(true);
      setTimeout(() => setCopiedSecret(false), 2000);
    }
  };

  return (
    <div className="panel endpoints-panel">
      <div className="panel-header-row">
        <div>
          <h3 className="panel-title">My Webhook Endpoints</h3>
          <p className="panel-subtitle">Endpoints configured exclusively for your user account</p>
        </div>
        <button
          className="btn-ghost btn-sm"
          onClick={fetchEndpoints}
          disabled={loading}
          title="Refresh endpoints"
        >
          {loading ? '⏳' : '↻ Refresh'}
        </button>
      </div>

      {/* ── Registration Form ── */}
      <form className="form endpoint-create-form" onSubmit={handleRegister}>
        <div className="input-group-row">
          <input
            id="reg-target-url"
            type="url"
            placeholder="https://example.com/webhook or http://localhost:9090/ok"
            value={targetUrl}
            onChange={(e) => setTargetUrl(e.target.value)}
            required
            className="flex-1"
          />
          <button
            id="register-btn"
            type="submit"
            className="btn-primary"
            disabled={registering}
          >
            {registering ? 'Registering…' : '+ Add Endpoint'}
          </button>
        </div>
      </form>

      {/* ── Registration Success Modal / Banner ── */}
      {regResult && (
        <div className="result-box result-success animate-fade">
          <div className="result-header">
            <strong>✓ Endpoint Registered Successfully!</strong>
            <button className="btn-close" onClick={() => setRegResult(null)}>✕</button>
          </div>
          <p className="mt-1">
            <strong>Endpoint ID:</strong>{' '}
            <code className="inline-code">{regResult.id}</code>
          </p>
          <div className="secret-card">
            <div className="secret-card-header">
              <span>HMAC-SHA256 Signing Secret (Shown Once)</span>
              <button
                type="button"
                className="btn-copy"
                onClick={() => copyToClipboard(regResult.secret_key, 'secret')}
              >
                {copiedSecret ? '✓ Copied' : '📋 Copy Secret'}
              </button>
            </div>
            <code className="secret-code">{regResult.secret_key}</code>
            <p className="secret-warning">
              ⚠ Store this secret key securely. The server signs each delivery with this key, but never displays it again.
            </p>
          </div>
        </div>
      )}

      {regError && <div className="result-box result-error">✗ {regError}</div>}
      {error && <div className="result-box result-error">✗ {error}</div>}

      {/* ── Endpoints List ── */}
      <div className="endpoints-list-wrapper">
        {loading && endpoints.length === 0 ? (
          <p className="text-muted text-sm py-4 text-center">Loading endpoints…</p>
        ) : endpoints.length === 0 ? (
          <div className="empty-endpoints">
            <span className="empty-icon">🔗</span>
            <p>You haven't created any webhook endpoints yet.</p>
            <p className="text-muted text-xs">Enter a target URL above to get your first HMAC signing endpoint.</p>
          </div>
        ) : (
          <div className="endpoint-cards-grid">
            {endpoints.map((ep) => {
              const isSelected = selectedEndpointId === ep.id;
              return (
                <div
                  key={ep.id}
                  className={`endpoint-card ${isSelected ? 'selected' : ''}`}
                  onClick={() => onSelectEndpoint && onSelectEndpoint(ep.id)}
                >
                  <div className="endpoint-card-top">
                    <div className="endpoint-url-badge" title={ep.target_url}>
                      <span className="status-indicator active"></span>
                      <span className="target-url-text">{ep.target_url}</span>
                    </div>
                    <div className="endpoint-card-actions">
                      <button
                        className="btn-ghost btn-xs text-danger"
                        onClick={(e) => handleDelete(ep.id, e)}
                        title="Delete this endpoint"
                      >
                        🗑
                      </button>
                    </div>
                  </div>

                  <div className="endpoint-card-bottom">
                    <div className="id-copy-row">
                      <span className="text-xs text-muted">ID:</span>
                      <code className="endpoint-id-snippet">{ep.id.slice(0, 13)}…</code>
                      <button
                        type="button"
                        className="btn-copy-xs"
                        onClick={(e) => {
                          e.stopPropagation();
                          copyToClipboard(ep.id, 'id', ep.id);
                        }}
                      >
                        {copiedId === ep.id ? '✓' : 'Copy'}
                      </button>
                    </div>

                    <button
                      type="button"
                      className={`btn-select-xs ${isSelected ? 'active' : ''}`}
                      onClick={(e) => {
                        e.stopPropagation();
                        onSelectEndpoint && onSelectEndpoint(ep.id);
                      }}
                    >
                      {isSelected ? '✓ Selected' : 'Select'}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
