import React, { useState } from 'react';
import ReplayButton from './ReplayButton';

const STATUS_COLORS = {
  SUCCESS: '#10b981',
  FAILED: '#ef4444',
  CIRCUIT_BROKEN: '#f59e0b',
  DLQ: '#a855f7',
  PENDING: '#64748b',
};

/**
 * LogRow — renders a single delivery attempt in the user's isolated log table.
 */
export default function LogRow({ log }) {
  const [expanded, setExpanded] = useState(false);
  const statusColor = STATUS_COLORS[log.status] || '#64748b';

  const formatTime = (ts) => {
    if (!ts) return 'Just now';
    try {
      const d = new Date(ts);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    } catch {
      return ts;
    }
  };

  return (
    <>
      <tr
        className="log-row"
        onClick={() => setExpanded((e) => !e)}
        style={{ cursor: 'pointer' }}
      >
        <td>
          <span className="status-badge" style={{ backgroundColor: `${statusColor}22`, color: statusColor, borderColor: statusColor }}>
            <span className="status-dot" style={{ backgroundColor: statusColor }}></span>
            {log.status}
          </span>
        </td>
        <td className="mono text-sm" title={log.event_id}>
          {log.event_id ? `${log.event_id.slice(0, 8)}…` : '—'}
        </td>
        <td className="text-sm url-cell" title={log.target_url}>
          {log.target_url || '—'}
        </td>
        <td className="text-center font-semibold">#{log.attempt_number ?? 1}</td>
        <td className="text-center">
          {log.http_status != null ? (
            <span className={log.http_status < 400 ? 'http-ok' : 'http-err'}>
              {log.http_status}
            </span>
          ) : (
            <span className="text-muted">—</span>
          )}
        </td>
        <td className="text-center">
          {log.execution_time_ms != null ? (
            <span className="latency-badge">{log.execution_time_ms}ms</span>
          ) : (
            '—'
          )}
        </td>
        <td className="text-muted text-xs">
          {formatTime(log.delivered_at)}
        </td>
        <td onClick={(e) => e.stopPropagation()}>
          <ReplayButton eventId={log.event_id} />
        </td>
      </tr>

      {expanded && (
        <tr className="log-row-detail">
          <td colSpan={8}>
            <div className="log-detail-card">
              <div className="log-detail-grid">
                <div>
                  <p className="detail-label">Full Event ID</p>
                  <p className="mono select-all">{log.event_id}</p>
                </div>
                <div>
                  <p className="detail-label">Endpoint ID</p>
                  <p className="mono select-all">{log.endpoint_id || '—'}</p>
                </div>
                <div>
                  <p className="detail-label">Target Destination</p>
                  <p className="mono break-all">{log.target_url || '—'}</p>
                </div>
                {log.delivered_at && (
                  <div>
                    <p className="detail-label">Timestamp</p>
                    <p className="mono">{new Date(log.delivered_at).toLocaleString()}</p>
                  </div>
                )}
              </div>

              {log.error_message && (
                <div className="log-detail-block error-block">
                  <p className="detail-label">Error Details</p>
                  <pre className="detail-error">{log.error_message}</pre>
                </div>
              )}

              {log.response_body && (
                <div className="log-detail-block">
                  <p className="detail-label">Response Body (Truncated)</p>
                  <pre className="detail-payload">{log.response_body}</pre>
                </div>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  );
}
