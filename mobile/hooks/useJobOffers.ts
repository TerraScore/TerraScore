import { useEffect, useRef, useState, useCallback } from 'react';
import { AppState } from 'react-native';
import { api } from '@/services/api';
import { getStoredToken } from '@/services/auth';
import type { Offer, WSOfferMessage, ApiResponse } from '@/types/api';

const WS_BASE = (process.env.EXPO_PUBLIC_API_URL ?? 'http://localhost:8080')
  .replace(/^http/, 'ws');

const RECONNECT_DELAY_MS = 3000;
const POLL_INTERVAL_MS = 15000; // Poll every 15s as fallback

// Global event listeners for cross-screen real-time updates
type EventCallback = (eventType: string, data: Record<string, string>) => void;
const eventListeners = new Set<EventCallback>();

export function onAgentEvent(cb: EventCallback) {
  eventListeners.add(cb);
  return () => { eventListeners.delete(cb); };
}

export function useJobOffers() {
  const [offers, setOffers] = useState<Offer[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeout = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pollInterval = useRef<ReturnType<typeof setInterval> | null>(null);

  const fetchOffers = useCallback(async () => {
    try {
      const { data } = await api.get<ApiResponse<Offer[]>>('/v1/agents/me/offers');
      setOffers(data.data ?? []);
    } catch {
      // ignore
    } finally {
      setIsLoading(false);
    }
  }, []);

  const removeOffer = useCallback((offerId: string) => {
    setOffers((prev) => prev.filter((o) => o.id !== offerId));
  }, []);

  const connectWS = useCallback(async () => {
    const token = await getStoredToken();
    if (!token) return;

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.onclose = null;
      wsRef.current.close();
    }

    const ws = new WebSocket(`${WS_BASE}/ws?token=${token}`);
    wsRef.current = ws;

    ws.onopen = () => {
      setIsConnected(true);
      // Clear polling when WebSocket is active
      if (pollInterval.current) {
        clearInterval(pollInterval.current);
        pollInterval.current = null;
      }
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);

        // Handle offer messages (from offers channel)
        if (msg.offer_id) {
          fetchOffers();
          return;
        }

        // Handle general events (from events channel)
        if (msg.type && msg.data) {
          // Notify all listeners (active jobs screen, etc.)
          eventListeners.forEach((cb) => cb(msg.type, msg.data));

          // Also refresh offers if job status changed
          if (msg.type === 'job.accepted' || msg.type === 'job.survey_submitted') {
            fetchOffers();
          }
        }
      } catch {
        // ignore malformed messages
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      wsRef.current = null;
      // Start polling as fallback
      if (!pollInterval.current) {
        pollInterval.current = setInterval(fetchOffers, POLL_INTERVAL_MS);
      }
      // Schedule reconnect
      reconnectTimeout.current = setTimeout(connectWS, RECONNECT_DELAY_MS);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [fetchOffers]);

  useEffect(() => {
    fetchOffers();
    connectWS();

    // Reconnect when app comes to foreground
    const sub = AppState.addEventListener('change', (state) => {
      if (state === 'active') {
        if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
          connectWS();
        }
        fetchOffers();
      }
    });

    return () => {
      sub.remove();
      if (wsRef.current) {
        wsRef.current.onclose = null;
        wsRef.current.close();
      }
      if (reconnectTimeout.current) clearTimeout(reconnectTimeout.current);
      if (pollInterval.current) clearInterval(pollInterval.current);
    };
  }, [connectWS, fetchOffers]);

  return { offers, isConnected, isLoading, refresh: fetchOffers, removeOffer };
}
