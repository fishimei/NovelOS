import { useEffect, useRef, useState } from 'react';

export type SseConnectionStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error';

export type SseMessage = {
  event: string;
  data: unknown;
};

const STORY_EVENTS = ['generation_step', 'plot_variable', 'character_turn', 'draft_delta', 'review_required'];

export function useEventSource(options: {
  url?: string;
  enabled: boolean;
  onMessage: (message: SseMessage) => void;
  onError?: (error: Event) => void;
}) {
  const [connectionStatus, setConnectionStatus] = useState<SseConnectionStatus>('idle');
  const onMessageRef = useRef(options.onMessage);
  const onErrorRef = useRef(options.onError);

  useEffect(() => {
    onMessageRef.current = options.onMessage;
    onErrorRef.current = options.onError;
  }, [options.onError, options.onMessage]);

  useEffect(() => {
    if (!options.enabled || !options.url) {
      setConnectionStatus('idle');
      return undefined;
    }

    setConnectionStatus('connecting');
    const source = new EventSource(options.url);

    const parseData = (data: string) => {
      if (!data) {
        return undefined;
      }

      try {
        return JSON.parse(data);
      } catch {
        return data;
      }
    };

    const handleEvent =
      (eventName: string): EventListener =>
      (event) => {
        const messageEvent = event as MessageEvent;
        onMessageRef.current({
          event: eventName,
          data: parseData(messageEvent.data),
        });
      };

    const listeners = STORY_EVENTS.map((eventName) => ({
      eventName,
      listener: handleEvent(eventName),
    }));

    source.onopen = () => setConnectionStatus('open');
    source.onerror = (event) => {
      setConnectionStatus('error');
      onErrorRef.current?.(event);
    };
    source.onmessage = handleEvent('message');

    listeners.forEach(({ eventName, listener }) => {
      source.addEventListener(eventName, listener);
    });

    return () => {
      listeners.forEach(({ eventName, listener }) => {
        source.removeEventListener(eventName, listener);
      });
      source.close();
      setConnectionStatus('closed');
    };
  }, [options.enabled, options.url]);

  return { connectionStatus };
}
