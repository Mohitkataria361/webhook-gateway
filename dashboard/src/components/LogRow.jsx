import React, { useState } from 'react';
import ReplayButton from './ReplayButton';

const STATUS_COLORS = {
  SUCCESS: '#22c55e',
  FAILED: '#ef4444',
  CIRCUIT_BROKEN: '#f59e0b',
  DLQ: '#8b5cf6',
  PENDING: '#64748b',
};

/**
 * LogRow — renders a single delivery attempt in the live log table.
 */
export default function LogRow({ log }) {
  const [expanded, setExpanded] = useState(false);
  const statusColor = STATUS_COLORS[log.status] || '#64748b';

  return (
    <>
      <tr
        className="log-row"
        onClick={() => setExpanded((e) => !e)}
        style={{ cursor: 'pointer' }}
      >
        <td>
          <span className="status-badge" style={{ background: statusColor }}>
            {log.status}
          </span>
        </td>
        <td className="mono text-sm">{log.event_id?.slice(0, 8)}…</td>
        <td className="text-sm">{log.target_url || '—'}</td>
        <td className="text-center">{log.attempt_number ?? '—'}</td>
        <td className="text-center">
          {log.http_status != null ? (
            <span className={log.http_status < 400 ? 'http-ok' : 'http-err'}>
              {log.http_status}
            </span>
          ) : '—'}
        </td>
        <td className="text-center">{log.execution_time_ms != null ? `${log.execution_time_ms}ms` : '—'}</td>
        <td>
          <ReplayButton eventId={log.event_id} />
        </td>
      </tr>

      {expanded && (
        <tr className="log-row-detail">
          <td colSpan={7}>
            <div className="log-detail-card">
              <div className="log-detail-grid">
                <div>
                  <p className="detail-label">Event ID</p>
                  <p className="mono">{log.event_id}</p>
                </div>
                <div>
                  <p className="detail-label">Endpoint ID</p>
                  <p className="mono">{log.endpoint_id}</p>
                </div>
                {log.error_message && (
                  <div>
                    <p className="detail-label">Error</p>
                    <p className="detail-error">{log.error_message}</p>
                  </div>
                )}
              </div>
            </div>
          </td>
        </tr>
      )}
    </>
  );
}
