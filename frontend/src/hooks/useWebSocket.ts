import { useEffect, useRef, useState, useCallback } from 'react';
import { API_BASE_URL } from '@/lib/config';

interface BuildStatusUpdate {
  type: string;
  app_id: string;
  build_job_id: string;
  status: 'building' | 'completed' | 'failed' | 'deploying';
  progress?: {
    stage: string;
    message: string;
    percent: number;
  };
  data?: Record<string, unknown>;
}

interface UseWebSocketOptions {
  appId: string;
  onMessage?: (update: BuildStatusUpdate) => void;
  enabled?: boolean;
}

export function useWebSocket({ appId, onMessage, enabled = true }: UseWebSocketOptions) {
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<BuildStatusUpdate | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);

  const connect = useCallback(() => {
    if (!enabled || !appId) return;

    // Convert API base URL to WebSocket URL
    let wsUrl: string;
    if (API_BASE_URL.startsWith('http://')) {
      wsUrl = API_BASE_URL.replace('http://', 'ws://');
    } else if (API_BASE_URL.startsWith('https://')) {
      wsUrl = API_BASE_URL.replace('https://', 'wss://');
    } else {
      // Fallback to same origin
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = window.location.host;
      wsUrl = `${protocol}//${host}`;
    }
    
    wsUrl = `${wsUrl}/ws/build-status?app_id=${appId}`;

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        setIsConnected(true);
        reconnectAttemptsRef.current = 0;
        console.log('WebSocket connected for app:', appId);
      };

      ws.onmessage = (event) => {
        try {
          const update: BuildStatusUpdate = JSON.parse(event.data);
          setLastMessage(update);
          if (onMessage) {
            onMessage(update);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      ws.onerror = (error) => {
        console.error('WebSocket error:', error);
      };

      ws.onclose = () => {
        setIsConnected(false);
        console.log('WebSocket disconnected for app:', appId);

        // Reconnect with exponential backoff
        if (enabled) {
          const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000);
          reconnectAttemptsRef.current += 1;
          
          reconnectTimeoutRef.current = setTimeout(() => {
            connect();
          }, delay);
        }
      };

      wsRef.current = ws;
    } catch (error) {
      console.error('Failed to create WebSocket connection:', error);
      setIsConnected(false);
    }
  }, [appId, enabled, onMessage]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    setIsConnected(false);
  }, []);

  useEffect(() => {
    if (enabled && appId) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [enabled, appId, connect, disconnect]);

  return {
    isConnected,
    lastMessage,
    connect,
    disconnect,
  };
}

