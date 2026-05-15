import { useMemo, useState } from 'react';

import { storyRunEventsUrl } from '../../api/storyRuns';
import { useEventSource, type SseMessage } from '../../hooks/useEventSource';

type StoryEventState = {
  draftText: string;
  generationSteps: unknown[];
  plotVariables: unknown[];
  characterTurns: unknown[];
  reviewItems: unknown[];
  rawEvents: SseMessage[];
};

const initialState: StoryEventState = {
  draftText: '',
  generationSteps: [],
  plotVariables: [],
  characterTurns: [],
  reviewItems: [],
  rawEvents: [],
};

function stringifyDelta(data: unknown) {
  if (typeof data === 'string') {
    return data;
  }

  if (data && typeof data === 'object' && 'text' in data) {
    return String((data as { text?: unknown }).text ?? '');
  }

  if (data && typeof data === 'object' && 'delta' in data) {
    return String((data as { delta?: unknown }).delta ?? '');
  }

  return '';
}

export function useStoryRunEvents(runId: string) {
  const [state, setState] = useState<StoryEventState>(initialState);
  const url = useMemo(() => (runId ? storyRunEventsUrl(runId) : undefined), [runId]);

  const { connectionStatus } = useEventSource({
    url,
    enabled: Boolean(runId),
    onMessage: (message) => {
      setState((current) => {
        const next = { ...current, rawEvents: [...current.rawEvents, message] };

        if (message.event === 'draft_delta') {
          next.draftText = current.draftText + stringifyDelta(message.data);
        }

        if (message.event === 'generation_step') {
          next.generationSteps = [...current.generationSteps, message.data];
        }

        if (message.event === 'plot_variable') {
          next.plotVariables = [...current.plotVariables, message.data];
        }

        if (message.event === 'character_turn') {
          next.characterTurns = [...current.characterTurns, message.data];
        }

        if (message.event === 'review_required') {
          next.reviewItems = [...current.reviewItems, message.data];
        }

        return next;
      });
    },
  });

  return {
    ...state,
    connectionStatus,
  };
}
