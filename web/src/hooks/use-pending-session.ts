import { useState, useEffect, useCallback } from 'react';
import { getTransport } from '@/lib/transport';
import type { NewSessionPendingEvent, SessionPendingCancelledEvent } from '@/lib/transport/types';

/**
 * Hook to listen for new_session_pending events
 * Used for force project binding feature
 */
export function usePendingSession() {
  const [pendingSession, setPendingSession] = useState<NewSessionPendingEvent | null>(null);

  useEffect(() => {
    const transport = getTransport();

    const unsubscribePending = transport.subscribe<NewSessionPendingEvent>(
      'new_session_pending',
      (event) => {
        setPendingSession(event);
      }
    );

    const unsubscribeCancelled = transport.subscribe<SessionPendingCancelledEvent>(
      'session_pending_cancelled',
      (event) => {
        // Close dialog if the cancelled session matches current pending session
        setPendingSession((current) => {
          if (current && current.sessionID === event.sessionID) {
            return null;
          }
          return current;
        });
      }
    );

    return () => {
      unsubscribePending();
      unsubscribeCancelled();
    };
  }, []);

  const clearPendingSession = useCallback(() => {
    setPendingSession(null);
  }, []);

  return {
    pendingSession,
    clearPendingSession,
  };
}
