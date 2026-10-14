import axios from 'axios';

// Resolve base API URL:
// 1. Explicit environment variable (VITE_API_URL)
// 2. Fallback for deployed Render app (webhook-gateway-1.onrender.com -> backend webhook-gateway-ciq7.onrender.com)
// 3. Fallback to '/api/v1' for local dev with Vite proxy or same-origin deployments
export const getApiBaseUrl = () => {
  const envUrl = import.meta.env.VITE_API_URL;
  if (envUrl) {
    return envUrl.replace(/\/+$/, '');
  }
  if (typeof window !== 'undefined' && window.location.hostname.includes('webhook-gateway-1.onrender.com')) {
    return 'https://webhook-gateway-ciq7.onrender.com/api/v1';
  }
  return '/api/v1';
};

export const getBackendBaseUrl = () => {
  const apiBase = getApiBaseUrl();
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    return apiBase.replace(/\/api\/v1\/?$/, '');
  }
  return '';
};

export const getOAuthUrl = (provider) => {
  const apiBase = getApiBaseUrl();
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    const normalized = apiBase.endsWith('/api/v1') ? apiBase : `${apiBase}/api/v1`;
    return `${normalized}/auth/${provider}`;
  }
  return `/api/v1/auth/${provider}`;
};

export const getWebSocketUrl = (token) => {
  if (import.meta.env.VITE_WS_URL) {
    return `${import.meta.env.VITE_WS_URL}?token=${encodeURIComponent(token)}`;
  }
  const backend = getBackendBaseUrl();
  if (backend) {
    const wsBase = backend.replace(/^http/, 'ws');
    return `${wsBase}/ws/logs?token=${encodeURIComponent(token)}`;
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/ws/logs?token=${encodeURIComponent(token)}`;
};

const api = axios.create({
  baseURL: getApiBaseUrl(),
  headers: { 'Content-Type': 'application/json' },
});

// Attach API Key or JWT Bearer token to all outgoing API requests
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('api_key') || localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Auto-handle 401 Unauthorized
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401 && !error.config.url.includes('/auth/keys')) {
      window.dispatchEvent(new Event('auth:unauthorized'));
    }
    return Promise.reject(error);
  }
);

// --- Auth & API Key Endpoints ---
export const getAuthConfig = () => api.get('/auth/config');
export const getMe = () => api.get('/auth/me');
export const devLogin = (username, email, name) =>
  api.post('/auth/dev-login', { username, email, name });

export const generateApiKey = (userId) =>
  api.post('/auth/keys', userId ? { user_id: userId } : {});

export const adminGenerateKey = (userId) => {
  const base = getBackendBaseUrl();
  return axios.post(`${base}/api/admin/generate-key`, userId ? { user_id: userId } : {});
};

// --- Endpoint / Hooks Endpoints ---
export const getEndpoints = () => api.get('/endpoints');
export const registerEndpoint = (targetUrl) =>
  api.post('/endpoints', { target_url: targetUrl });
export const deleteEndpoint = (id) => api.delete(`/endpoints/${id}`);

// --- Webhook Dispatch ---
export const sendWebhook = ({ endpointId, eventType, payload, idempotencyKey }) =>
  api.post('/webhooks/send', {
    endpoint_id: endpointId,
    event_type: eventType,
    payload,
    idempotency_key: idempotencyKey,
  });

// --- Delivery Logs & Replay ---
export const getLogs = () => api.get('/logs');
export const getDeliveries = (eventId) => api.get(`/events/${eventId}/deliveries`);
export const replayEvent = (eventId) => api.post(`/events/${eventId}/replay`);
