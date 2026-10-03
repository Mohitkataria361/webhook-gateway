import React, { useState } from 'react';
import { replayEvent } from '../api/webhookApi';

/**
 * ReplayButton — renders a pill button that re-queues an event via the API.
 * Shows spinner during request and success/error state briefly.
 */
export default function ReplayButton({ eventId }) {
  const [state, setState] = useState('idle'); // idle | loading | success | error

  const handleReplay = async () => {
    setState('loading');
    try {
      await replayEvent(eventId);
      setState('success');
      setTimeout(() => setState('idle'), 2500);
    } catch (err) {
      console.error('[replay] Error:', err);
      setState('error');
      setTimeout(() => setState('idle'), 2500);
    }
  };

  const labels = {
    idle: '↺ Replay',
    loading: '⏳ Re-queuing…',
    success: '✓ Re-queued!',
    error: '✗ Failed',
  };

  const classes = {
    idle: 'replay-btn replay-btn--idle',
    loading: 'replay-btn replay-btn--loading',
    success: 'replay-btn replay-btn--success',
    error: 'replay-btn replay-btn--error',
  };

  return (
    <button
      id={`replay-${eventId}`}
      className={classes[state]}
      onClick={handleReplay}
      disabled={state === 'loading'}
      title={`Replay webhook event ${eventId}`}
    >
      {labels[state]}
    </button>
  );
}
