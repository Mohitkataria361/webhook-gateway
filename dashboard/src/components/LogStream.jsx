import React from 'react';
import LogRow from './LogRow';
import { useWebSocket } from '../hooks/useWebSocket';

/**
 * LogStream — live delivery log table streaming from the WebSocket hub.
 */
export default function LogStream() {
  const { logs, connected, error, clearLogs } = useWebSocket();

  return (
    <section className="log-stream-section">
      <div className="log-stream-header">
        <div className="header-left">
          <h2>Live Delivery Log</h2>
          <span className={`connection-badge ${connected ? 'conn-live' : 'conn-offline'}`}>
            <span className={`conn-dot ${connected ? 'dot-live' : 'dot-offline'}`}></span>
            {connected ? 'Connected' : 'Reconnecting…'}
          </span>
        </div>
        <div className="header-right">
          <span className="log-count">{logs.length} events</span>
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
              <th>Action</th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 ? (
              <tr>
                <td colSpan={7} className="empty-state">
                  {connected
                    ? '⏳ Waiting for webhook events…'
                    : '🔌 Connecting to live stream…'}
                </td>
              </tr>
            ) : (
              logs.map((log) => <LogRow key={log.id} log={log} />)
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
