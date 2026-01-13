import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useSourceCRUD } from '../sources';
import { StatusType } from '@odigos/ui-kit/types';
import { StatusKeys, useStatusStore } from '@/store';
import { useDestinationCRUD } from '../destinations';
import { NotificationIcon } from '@odigos/ui-kit/icons';
import { type NotifyPayload, useNotificationStore, useProgressStore, ProgressKeys } from '@odigos/ui-kit/store';

enum EventTypes {
  CONNECTED = 'CONNECTED',
  ADDED = 'Added',
  MODIFIED = 'Modified',
  DELETED = 'Deleted',
}

enum CrdTypes {
  InstrumentationConfig = 'InstrumentationConfig',
  Destination = 'Destination',
}

const EVENT_DEBOUNCE_MS = 5000;

export const useSSE = () => {
  const { fetchSources } = useSourceCRUD();
  const { setStatusStore } = useStatusStore();
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();

  const clearStatusMessage = () => setStatusStore(StatusKeys.Instrumentation, undefined);

  const maxRetries = 10;
  const retryCount = useRef(0);

  const eventsRef = useRef<
    | {
        [eventType in EventTypes]?: {
          handler: NodeJS.Timeout | null;
          timestamp: number | null;
        };
      }
    | null
  >(null);

  const resetEventHandler = (eventType: EventTypes) => {
    if (eventsRef.current && eventsRef.current[eventType]) {
      if (eventsRef.current?.[eventType]?.handler) {
        clearTimeout(eventsRef.current[eventType]!.handler as NodeJS.Timeout);
        eventsRef.current[eventType]!.handler = null;
      }
      eventsRef.current[eventType]!.timestamp = null;
    }
  };

  const handleEvent = (eventType: EventTypes, successCallback: () => void) => {
    if (!eventsRef.current) {
      eventsRef.current = {
        [eventType]: {
          handler: null,
          timestamp: null,
        },
      };
    } else if (!eventsRef.current[eventType]) {
      eventsRef.current[eventType] = {
        handler: null,
        timestamp: null,
      };
    }

    if (eventsRef.current![eventType]!.handler) clearTimeout(eventsRef.current![eventType]!.handler as NodeJS.Timeout);
    eventsRef.current![eventType]!.timestamp = Date.now();

    // if last message was over `EVENT_DEBOUNCE_MS` time ago - fetch all sources.
    // once the condition is met, or a new event is received, the handler is cleared.
    eventsRef.current[eventType]!.handler = setTimeout(() => {
      resetEventHandler(eventType);
      successCallback();
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
        const notification: NotifyPayload = {
          type: data.type,
          title: data.event || '',
          message: data.data || '',
          crdType: data.crdType || '',
          target: data.target,
        };

        const isConnected = [EventTypes.CONNECTED].includes(notification.crdType);
        const isSource = [CrdTypes.InstrumentationConfig].includes(notification.crdType);
        const isDestination = [CrdTypes.Destination].includes(notification.crdType);

        // do not notify for: connected, modified events, or sources
        if ((notification.title || notification.message) && !isConnected && notification.title !== EventTypes.MODIFIED && !isSource) {
          addNotification(notification);
        }

        // Handle specific CRD types
        if (isSource) {
          switch (notification.title) {
            case EventTypes.ADDED:
              setStatusStore(StatusKeys.Instrumentation, { status: StatusType.Warning, label: 'Creating sources, please wait a moment...', leftIcon: NotificationIcon });

              const newCreated = Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              useProgressStore.getState().addProgress(ProgressKeys.Instrumenting, newCreated);

              handleEvent(EventTypes.ADDED, () => {
                const { progress, resetProgress } = useProgressStore.getState();
                addNotification({ type: StatusType.Success, title: EventTypes.ADDED, message: `Successfully created ${progress[ProgressKeys.Instrumenting]?.total} sources` });
                setStatusStore(StatusKeys.Instrumentation, { status: StatusType.Warning, label: 'Instrumenting sources, please wait a moment...', leftIcon: NotificationIcon });
                resetProgress(ProgressKeys.Instrumenting);
              });
              break;

            case EventTypes.MODIFIED:
              const { progress } = useProgressStore.getState();
              if (!progress[ProgressKeys.Instrumenting] && !progress[ProgressKeys.Uninstrumenting]) {
                handleEvent(EventTypes.MODIFIED, () => {
                  clearStatusMessage();
                  fetchSources();
                });
              }
              break;

            case EventTypes.DELETED:
              const newDeleted = Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              useProgressStore.getState().addProgress(ProgressKeys.Uninstrumenting, newDeleted);

              handleEvent(EventTypes.DELETED, () => {
                const { progress, resetProgress } = useProgressStore.getState();
                addNotification({ type: StatusType.Success, title: EventTypes.DELETED, message: `Successfully deleted ${progress[ProgressKeys.Uninstrumenting]?.total} sources` });
                resetProgress(ProgressKeys.Uninstrumenting);
                clearStatusMessage();

                fetchSources();
              });
              break;

            default:
              break;
          }
        } else if (isDestination) {
          fetchDestinations();
        } else {
          console.warn('Unhandled SSE for CRD type:', notification.crdType);
        }

        // Reset retry count on successful connection
        retryCount.current = 0;
      };

      return es;
    };

    // Initialize event source connection
    const es = connect();
    // Clean up event source on component unmount
    return () => {
      es?.close();
      Object.values(EventTypes).forEach((eventType) => resetEventHandler(eventType));
      clearStatusMessage();
    };
  }, []);
};
