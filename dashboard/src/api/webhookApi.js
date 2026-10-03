import axios from 'axios';

const api = axios.create({
  baseURL: '/api/v1',
  headers: { 'Content-Type': 'application/json' },
});

/**
 * Register a new webhook endpoint.
 * @param {string} userId - UUID of the user
 * @param {string} targetUrl - The destination URL
 */
export const registerEndpoint = (userId, targetUrl) =>
  api.post('/endpoints', { user_id: userId, target_url: targetUrl });

/**
 * Send a webhook event to a registered endpoint.
 */
export const sendWebhook = ({ endpointId, eventType, payload, idempotencyKey }) =>
  api.post('/webhooks/send', {
    endpoint_id: endpointId,
    event_type: eventType,
    payload,
    idempotency_key: idempotencyKey,
  });

/**
 * Get all delivery attempts for a specific event.
 * @param {string} eventId - UUID of the webhook event
 */
export const getDeliveries = (eventId) => api.get(`/events/${eventId}/deliveries`);

/**
 * Replay (re-queue) an existing event.
 * @param {string} eventId - UUID of the webhook event to replay
 */
export const replayEvent = (eventId) => api.post(`/events/${eventId}/replay`);
