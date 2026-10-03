/**
 * useWebSocket.js
 * Custom hook that opens a WebSocket connection and streams
 * delivery log events from the Go backend hub.
 */
import { useState, useEffect, useCallback, useRef } from 'react';

const WS_URL = `ws://${window.location.host}/ws/logs`;

export function useWebSocket() {
  const [logs, setLogs] = useState([]);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState(null);
  const wsRef = useRef(null);
  const reconnectTimer = useRef(null);

  const connect = useCallback(() => {
    const ws = new WebSocket(WS_URL);
    wsRef.current = ws;

    ws.onopen = () => {
      setConnected(true);
      setError(null);
      clearTimeout(reconnectTimer.current);
    };

    ws.onmessage = (event) => {
      try {
        const log = JSON.parse(event.data);
        // Prepend newest logs so they appear at the top.
        setLogs((prev) => [{ ...log, id: Date.now() + Math.random() }, ...prev].slice(0, 200));
      } catch (e) {
        console.warn('[ws] Failed to parse message:', e);
      }
    };

    ws.onerror = (e) => {
      setError('WebSocket error — check that the API server is running.');
      setConnected(false);
    };

    ws.onclose = () => {
      setConnected(false);
      // Auto-reconnect after 3 seconds.
      reconnectTimer.current = setTimeout(connect, 3000);
    };
  }, []);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
    };
  }, [connect]);

  const clearLogs = useCallback(() => setLogs([]), []);

  return { logs, connected, error, clearLogs };
}
