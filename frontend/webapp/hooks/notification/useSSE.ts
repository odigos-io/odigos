import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useStatusStore } from '@/store';
import { useSourceCRUD } from '../sources';
import { StatusType } from '@odigos/ui-kit/types';
import { useDestinationCRUD } from '../destinations';
import { NotificationIcon } from '@odigos/ui-kit/icons';
import { type NotifyPayload, useInstrumentStore, useNotificationStore } from '@odigos/ui-kit/store';

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
  const { setInstrumentAwait, setInstrumentCount } = useInstrumentStore();

  const clearStatusMessage = () => {
    const { priorityMessage } = useStatusStore.getState();
    if (!priorityMessage) setStatusStore({ status: StatusType.Default, message: '', leftIcon: undefined });
  };

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
          const { isAwaitingInstrumentation } = useInstrumentStore.getState();
          const { priorityMessage, message } = useStatusStore.getState();

          switch (notification.title) {
            case EventTypes.ADDED:
              if (!isAwaitingInstrumentation) setInstrumentAwait(true);

              const statusMessage1 = 'Creating sources, please wait a moment...';
              if (!priorityMessage && message !== statusMessage1) setStatusStore({ status: StatusType.Warning, message: statusMessage1, leftIcon: NotificationIcon });

              const { sourcesCreated } = useInstrumentStore.getState();
              const totalCreated = sourcesCreated + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesCreated', totalCreated);

              handleEvent(EventTypes.ADDED, () => {
                const statusMessage2 = 'Instrumenting sources, please wait a moment...';
                if (!priorityMessage && message !== statusMessage2) setStatusStore({ status: StatusType.Warning, message: statusMessage2, leftIcon: NotificationIcon });
                addNotification({ type: StatusType.Success, title: EventTypes.ADDED, message: `Successfully created ${totalCreated} sources` });

                fetchSources();

                setInstrumentAwait(false);
                setInstrumentCount('sourcesToCreate', 0);
                setInstrumentCount('sourcesCreated', 0);
              });
              break;

            case EventTypes.MODIFIED:
              if (!isAwaitingInstrumentation) {
                handleEvent(EventTypes.MODIFIED, () => {
                  clearStatusMessage();
                  fetchSources();
                });
              }
              break;

            case EventTypes.DELETED:
              if (!isAwaitingInstrumentation) setInstrumentAwait(true);

              const { sourcesDeleted } = useInstrumentStore.getState();
              const totalDeleted = sourcesDeleted + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesDeleted', totalDeleted);

              handleEvent(EventTypes.DELETED, () => {
                clearStatusMessage();
                addNotification({ type: StatusType.Success, title: EventTypes.DELETED, message: `Successfully deleted ${totalDeleted} sources` });

                fetchSources();

                setInstrumentAwait(false);
                setInstrumentCount('sourcesToDelete', 0);
                setInstrumentCount('sourcesDeleted', 0);
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
