/**
 * useWebSocket.js
 * Custom hook that connects to the authenticated WebSocket hub
 * and streams delivery log events isolated to the current user.
 */
import { useState, useEffect, useCallback, useRef } from 'react';
import { getLogs, getWebSocketUrl } from '../api/webhookApi';

const getWsUrl = (token) => getWebSocketUrl(token);

export function useWebSocket(token) {
  const [logs, setLogs] = useState([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const [loadingHistory, setLoadingHistory] = useState(false);
  const wsRef = useRef(null);
  const reconnectTimer = useRef(null);

  // Load user's initial log history from REST API
  const refreshHistory = useCallback(async () => {
    if (!token) return;
    setLoadingHistory(true);
    try {
      const { data } = await getLogs();
      if (data?.logs) {
        setLogs(data.logs);
      }
    } catch (err) {
      console.warn('[ws] Failed to load log history:', err);
    } finally {
      setLoadingHistory(false);
    }
  }, [token]);

  const connect = useCallback(() => {
    if (!token) {
      setConnected(false);
      return;
    }

    try {
      const ws = new WebSocket(getWsUrl(token));
      wsRef.current = ws;

      ws.onopen = () => {
        setConnected(true);
        setError(null);
        clearTimeout(reconnectTimer.current);
      };

      ws.onmessage = (event) => {
        try {
          const newLog = JSON.parse(event.data);
          // Prepend newest logs so they appear at the top.
          setLogs((prev) => {
            // Deduplicate if existing ID
            const exists = prev.some((l) => l.id && l.id === newLog.id);
            if (exists) {
              return prev.map((l) => (l.id === newLog.id ? newLog : l));
            }
            return [{ ...newLog, id: newLog.id || `live-${Date.now()}-${Math.random()}` }, ...prev].slice(0, 200);
          });
        } catch (e) {
          console.warn('[ws] Failed to parse message:', e);
        }
      };

      ws.onerror = (e) => {
        setError('WebSocket disconnected. Reconnecting…');
        setConnected(false);
      };

      ws.onclose = () => {
        setConnected(false);
        // Auto-reconnect after 3 seconds if token is still valid
        reconnectTimer.current = setTimeout(() => {
          if (token) connect();
        }, 3000);
      };
    } catch (e) {
      setError('Failed to initialize WebSocket');
    }
  }, [token]);

  useEffect(() => {
    if (token) {
      refreshHistory();
      connect();
    } else {
      setLogs([]);
      setConnected(false);
      wsRef.current?.close();
    }

    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [token, connect, refreshHistory]);

  const clearLogs = useCallback(() => setLogs([]), []);

  return { logs, connected, error, loadingHistory, refreshHistory, clearLogs };
}
