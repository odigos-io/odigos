import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { EntityTypes, StatusType, type WorkloadId } from '@odigos/ui-kit/types';
import { getIdFromSseTarget, safeJsonParse } from '@odigos/ui-kit/functions';
import { useEntityStore, useNotificationStore, useProgressStore, ProgressKeys } from '@odigos/ui-kit/store';

enum EventTypes {
  CONNECTED = 'CONNECTED',
  ADDED = 'Added',
  MODIFIED = 'Modified',
  DELETED = 'Deleted',
  DiagnoseStagesRequested = 'DiagnoseStagesRequested',
  DiagnoseStageCompleted = 'DiagnoseStageCompleted',
}

enum CrdTypes {
  InstrumentationConfig = 'InstrumentationConfig',
  Destination = 'Destination',
}

interface DebouncedEvent {
  handler: NodeJS.Timeout | null;
  targets: string[];
}

const EVENT_DEBOUNCE_MS = 5000;

export const useSSE = () => {
  const { fetchSources, fetchSourcesByTargets } = useSourceCRUD();
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { removeEntities } = useEntityStore();

  const maxRetries = 10;
  const retryCount = useRef(0);

  const eventsRef = useRef<Partial<Record<EventTypes, DebouncedEvent>> | null>(null);

  const resetEventHandler = (eventType: EventTypes) => {
    if (eventsRef.current?.[eventType]) {
      if (eventsRef.current[eventType]!.handler) {
        clearTimeout(eventsRef.current[eventType]!.handler as NodeJS.Timeout);
        eventsRef.current[eventType]!.handler = null;
      }
      eventsRef.current[eventType]!.targets = [];
    }
  };

  const handleEvent = (eventType: EventTypes, targets: string[], successCallback: (accumulatedTargets: string[]) => void) => {
    if (!eventsRef.current) {
      eventsRef.current = {};
    }
    if (!eventsRef.current[eventType]) {
      eventsRef.current[eventType] = { handler: null, targets: [] };
    }

    const entry = eventsRef.current[eventType]!;

    const existing = new Set(entry.targets);
    for (const t of targets) {
      if (t && !existing.has(t)) {
        entry.targets.push(t);
      }
    }

    if (entry.handler) clearTimeout(entry.handler as NodeJS.Timeout);

    entry.handler = setTimeout(() => {
      const accumulated = [...entry.targets];
      resetEventHandler(eventType);
      successCallback(accumulated);
    }, EVENT_DEBOUNCE_MS);
  };

  useEffect(() => {
    const connect = () => {
      const es = new EventSource(API.EVENTS);

      es.onerror = () => {
        es.close();

        if (retryCount.current < maxRetries) {
          retryCount.current += 1;
          console.warn(`Disconnected from the server. Retrying connection (${retryCount.current})`);

          setTimeout(() => connect(), Math.min(10000, 1000 * Math.pow(2, retryCount.current)));
        } else {
          console.error(`Connection lost on ${new Date().toLocaleString()}. Please reboot the application`);
        }
      };

      es.onmessage = (event) => {
        const data = JSON.parse(event.data);
        const targets: string[] = data.targets || [];

        const isSource = data.crdType === CrdTypes.InstrumentationConfig;
        const isDestination = data.crdType === CrdTypes.Destination;
        const isDiagnoseRequested = data.event === EventTypes.DiagnoseStagesRequested;
        const isDiagnoseCompleted = data.event === EventTypes.DiagnoseStageCompleted;

        if (isSource) {
          switch (data.event) {
            case EventTypes.ADDED:
              const newCreated = Number(data.data?.toString().replace(/[^\d]/g, '') || 0);
              useProgressStore.getState().addProgress(ProgressKeys.Instrumenting, newCreated);

              handleEvent(data.event, targets, (accumulatedTargets) => {
                const { resetProgress } = useProgressStore.getState();
                resetProgress(ProgressKeys.Instrumenting);

                if (accumulatedTargets.length > 0) {
                  fetchSourcesByTargets(accumulatedTargets);
                } else {
                  fetchSources();
                }
              });
              break;

            case EventTypes.MODIFIED:
              const { progress } = useProgressStore.getState();
              if (!progress[ProgressKeys.Instrumenting] && !progress[ProgressKeys.Uninstrumenting]) {
                handleEvent(data.event, targets, (accumulatedTargets) => {
                  if (accumulatedTargets.length > 0) {
                    fetchSourcesByTargets(accumulatedTargets);
                  } else {
                    fetchSources();
                  }
                });
              }
              break;

            case EventTypes.DELETED:
              const newDeleted = Number(data.data?.toString().replace(/[^\d]/g, '') || 0);
              useProgressStore.getState().addProgress(ProgressKeys.Uninstrumenting, newDeleted);

              handleEvent(data.event, targets, (accumulatedTargets) => {
                const { resetProgress } = useProgressStore.getState();
                resetProgress(ProgressKeys.Uninstrumenting);

                if (accumulatedTargets.length > 0) {
                  const ids = accumulatedTargets.map((t) => getIdFromSseTarget(t, EntityTypes.Source) as WorkloadId).filter((id) => id.namespace && id.name && id.kind);
                  removeEntities(EntityTypes.Source, ids);
                } else {
                  fetchSources();
                }
              });
              break;

            default:
              break;
          }
        } else if (isDestination) {
          // only notify for: destination created/deleted
          if (data.event === EventTypes.ADDED || data.event === EventTypes.DELETED) {
            addNotification({ type: data.type, title: data.event, message: data.data, crdType: data.crdType });
          }
          handleEvent(data.event, targets, () => {
            fetchDestinations();
          });
        } else if (isDiagnoseRequested) {
          const events = safeJsonParse<string[]>(data.data, []);

          useProgressStore.getState().setProgress(ProgressKeys.DownloadingDiagnose, {
            total: events.length,
            current: 0,
            percentage: 0,
            list: events.map((event) => ({ label: event, status: StatusType.Default })),
          });
        } else if (isDiagnoseCompleted) {
          const { stage, status, message } = safeJsonParse<{ stage: string; status: StatusType; message?: string }>(data.data, { stage: '', status: StatusType.Success });

          if (status === StatusType.Success) {
            useProgressStore.getState().addProgress(ProgressKeys.DownloadingDiagnose, 1, { label: stage, status });
          } else {
            useProgressStore.getState().addProgress(ProgressKeys.DownloadingDiagnose, 0, { label: stage, status });
            addNotification({ type: StatusType.Error, title: 'Diagnose failed', message, crdType: '' });
          }
        } else {
          console.warn('Unhandled SSE for CRD type:', data.crdType);
        }

        retryCount.current = 0;
      };

      return es;
    };

    const es = connect();
    return () => {
      es?.close();
      Object.values(EventTypes).forEach((eventType) => resetEventHandler(eventType));
    };
  }, []);
};
