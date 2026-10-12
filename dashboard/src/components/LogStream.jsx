import React, { useState, useMemo } from 'react';
import LogRow from './LogRow';
import { useWebSocket } from '../hooks/useWebSocket';

/**
 * LogStream — isolated delivery log table streaming from the WebSocket hub.
 */
export default function LogStream({ token, user }) {
  const { logs, connected, error, loadingHistory, refreshHistory, clearLogs } = useWebSocket(token);
  const [filterStatus, setFilterStatus] = useState('ALL');
  const [searchTerm, setSearchTerm] = useState('');

  const filteredLogs = useMemo(() => {
    return logs.filter((item) => {
      if (filterStatus !== 'ALL' && item.status !== filterStatus) {
        return false;
      }
      if (searchTerm.trim()) {
        const term = searchTerm.toLowerCase();
        const matchesEvent = item.event_id?.toLowerCase().includes(term);
        const matchesUrl = item.target_url?.toLowerCase().includes(term);
        const matchesEndpoint = item.endpoint_id?.toLowerCase().includes(term);
        if (!matchesEvent && !matchesUrl && !matchesEndpoint) {
          return false;
        }
      }
      return true;
    });
  }, [logs, filterStatus, searchTerm]);

  const statuses = ['ALL', 'SUCCESS', 'FAILED', 'CIRCUIT_BROKEN', 'DLQ'];

  return (
    <section className="log-stream-section">
      <div className="log-stream-header">
        <div className="header-left">
          <div className="title-row">
            <h2>Live Delivery Log</h2>
            <span className="isolated-badge" title="Deliveries are isolated to your account">
              🔒 Private Stream ({user?.email || 'Authenticated'})
            </span>
          </div>
          <span className={`connection-badge ${connected ? 'conn-live' : 'conn-offline'}`}>
            <span className={`conn-dot ${connected ? 'dot-live' : 'dot-offline'}`}></span>
            {connected ? 'WebSocket Live' : 'Reconnecting…'}
          </span>
        </div>

        <div className="header-right">
          <button
            className="btn-ghost"
            onClick={refreshHistory}
            disabled={loadingHistory}
            title="Reload delivery history"
          >
            {loadingHistory ? '⏳ Loading…' : '↻ Refresh'}
          </button>
          <button id="clear-logs-btn" className="clear-btn" onClick={clearLogs}>
            Clear
          </button>
        </div>
      </div>

      {error && (
        <div className="error-banner">
          ⚠ {error}
        </div>
      )}

      {/* ── Filters & Search ── */}
      <div className="log-filters-bar">
        <div className="filter-pills">
          {statuses.map((st) => (
            <button
              key={st}
              className={`filter-pill ${filterStatus === st ? 'active' : ''}`}
              onClick={() => setFilterStatus(st)}
            >
              {st}
            </button>
          ))}
        </div>
        <div className="search-box">
          <input
            type="text"
            placeholder="Search by Event ID or URL…"
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />
          {searchTerm && (
            <button className="search-clear" onClick={() => setSearchTerm('')}>✕</button>
          )}
        </div>
      </div>

      <div className="table-wrapper">
        <table className="log-table">
          <thead>
            <tr>
              <th>Status</th>
              <th>Event ID</th>
              <th>Target URL</th>
              <th>Attempt</th>
              <th>HTTP</th>
              <th>Latency</th>
              <th>Time</th>
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {filteredLogs.length === 0 ? (
              <tr>
                <td colSpan={8} className="empty-state">
                  {loadingHistory
                    ? '⏳ Loading delivery history…'
                    : logs.length === 0
                    ? connected
                      ? '⚡ Ready! Dispatch a webhook above to inspect live deliveries.'
                      : '🔌 Connecting to your private live stream…'
                    : 'No delivery logs match your filter.'}
                </td>
              </tr>
            ) : (
              filteredLogs.map((log) => <LogRow key={log.id || log.event_id} log={log} />)
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
