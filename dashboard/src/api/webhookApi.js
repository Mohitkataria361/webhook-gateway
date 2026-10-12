import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
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
  const base = import.meta.env.VITE_API_URL
    ? import.meta.env.VITE_API_URL.replace(/\/api\/v1\/?$/, '')
    : '';
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
