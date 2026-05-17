// 故事 run 专用 SSE 状态 reducer。把原始事件整理成页面要展示的正文、审校项和过程信息。
import { useEffect, useMemo, useState } from 'react';

import { storyRunEventsUrl } from '../../api/storyRuns';
import { useEventSource, type SseMessage } from '../../hooks/useEventSource';
import type {
  StoryCharacterTurnEvent,
  StoryDraftDeltaEvent,
  StoryGenerationStepEvent,
  StoryOrchestrationStartedEvent,
  StoryPlotVariable,
} from '../../types/api';

type StoryEventState = {
  legacyDraftText: string;
  orchestrationStarts: StoryOrchestrationStartedEvent[];
  generationSteps: StoryGenerationStepEvent[];
  plotVariables: StoryPlotVariable[];
  characterTurns: StoryCharacterTurnEvent[];
  reviewItems: unknown[];
  rawEvents: SseMessage[];
};

const initialState: StoryEventState = {
  legacyDraftText: '',
  orchestrationStarts: [],
  generationSteps: [],
  plotVariables: [],
  characterTurns: [],
  reviewItems: [],
  rawEvents: [],
};

function stringifyDelta(data: unknown) {
  // 事件 payload schema 还未完全稳定，因此兼容常见的正文增量形态。
  if (typeof data === 'string') {
    return data;
  }

  if (data && typeof data === 'object' && 'content' in data) {
    return String((data as StoryDraftDeltaEvent).content ?? '');
  }

  if (data && typeof data === 'object' && 'text' in data) {
    return String((data as StoryDraftDeltaEvent).text ?? '');
  }

  if (data && typeof data === 'object' && 'delta' in data) {
    return String((data as StoryDraftDeltaEvent).delta ?? '');
  }

  return '';
}

function asGenerationStep(data: unknown): StoryGenerationStepEvent {
  return data && typeof data === 'object' ? (data as StoryGenerationStepEvent) : { step: String(data ?? '') };
}

function asPlotVariable(data: unknown): StoryPlotVariable {
  return data && typeof data === 'object' ? (data as StoryPlotVariable) : {};
}

function asOrchestrationStart(data: unknown): StoryOrchestrationStartedEvent {
  return data && typeof data === 'object' ? (data as StoryOrchestrationStartedEvent) : {};
}

function asCharacterTurn(data: unknown): StoryCharacterTurnEvent {
  return data && typeof data === 'object' ? (data as StoryCharacterTurnEvent) : {};
}

export function useStoryRunEvents(runId: string) {
  const [state, setState] = useState<StoryEventState>(initialState);
  const url = useMemo(() => (runId ? storyRunEventsUrl(runId) : undefined), [runId]);

  useEffect(() => {
    setState(initialState);
  }, [runId]);

  const { connectionStatus } = useEventSource({
    url,
    enabled: Boolean(runId),
    onMessage: (message) => {
      setState((current) => {
        // 始终保留原始事件，方便后续 Run Inspector 展示尚未建模的 payload。
        const next = { ...current, rawEvents: [...current.rawEvents, message] };

        if (message.event === 'draft_delta') {
          next.legacyDraftText = current.legacyDraftText + stringifyDelta(message.data);
        }

        if (message.event === 'story_orchestration_started') {
          next.orchestrationStarts = [...current.orchestrationStarts, asOrchestrationStart(message.data)];
        }

        if (message.event === 'generation_step') {
          next.generationSteps = [...current.generationSteps, asGenerationStep(message.data)];
        }

        if (message.event === 'plot_variable') {
          next.plotVariables = [...current.plotVariables, asPlotVariable(message.data)];
        }

        if (message.event === 'character_turn') {
          next.characterTurns = [...current.characterTurns, asCharacterTurn(message.data)];
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
