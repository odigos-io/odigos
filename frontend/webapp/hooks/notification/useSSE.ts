import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { getIdFromSseTarget } from '@odigos/ui-kit/functions';
import { EntityTypes, StatusType, type WorkloadId } from '@odigos/ui-kit/types';
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

const MODIFIED_DEBOUNCE_MS = 5000;
const MAX_EVENTS_FOR_SINGLE_FETCH = 10;

export const useSSE = () => {
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { fetchSources, fetchSourceById } = useSourceCRUD();

  const maxRetries = 10;
  const retryCount = useRef(0);

  const lastModifiedEventIds = useRef<WorkloadId[]>([]);
  const lastModifiedEventTimestamp = useRef<number | null>(null);
  const lastModifiedEventInterval = useRef<NodeJS.Timeout | null>(null);

  const resetLastModifiedEventRefs = () => {
    lastModifiedEventIds.current = [];
    lastModifiedEventTimestamp.current = null;
    if (lastModifiedEventInterval.current) clearInterval(lastModifiedEventInterval.current);
    lastModifiedEventInterval.current = null;
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

        const { setInstrumentAwait, isAwaitingInstrumentation, setInstrumentCount, sourcesToCreate, sourcesCreated, sourcesToDelete, sourcesDeleted } = useInstrumentStore.getState();

        const isConnected = [EventTypes.CONNECTED].includes(notification.crdType);
        const isSource = [CrdTypes.InstrumentationConfig].includes(notification.crdType);
        const isDestination = [CrdTypes.Destination].includes(notification.crdType);

        // do not notify for: connected, modified events, or sources that are still being instrumented
        if (!isConnected && !(isSource && isAwaitingInstrumentation) && notification.title !== EventTypes.MODIFIED) {
          addNotification(notification);
        }

        // Handle specific CRD types
        if (isSource) {
          switch (notification.title) {
            case EventTypes.MODIFIED:
              if (!isAwaitingInstrumentation && notification.target) {
                if (lastModifiedEventInterval.current) clearInterval(lastModifiedEventInterval.current);
                lastModifiedEventIds.current.push(getIdFromSseTarget(notification.target, EntityTypes.Source));
                lastModifiedEventTimestamp.current = Date.now();

                // if last message was over `MODIFIED_DEBOUNCE_MS` seconds ago, fetch the sources (all, or if less than `MAX_EVENTS_FOR_SINGLE_FETCH` then fetch each by id)...
                // the interval is to run a timestamp check every 1 second - once the condition is met, the interval is cleared.
                lastModifiedEventInterval.current = setInterval(async () => {
                  const timeSinceLastModified = Date.now() - (lastModifiedEventTimestamp.current || 0);

                  if (timeSinceLastModified > MODIFIED_DEBOUNCE_MS) {
                    if (lastModifiedEventIds.current.length <= MAX_EVENTS_FOR_SINGLE_FETCH) {
                      await Promise.allSettled(lastModifiedEventIds.current.map((id) => fetchSourceById(id)));
                    } else {
                      await fetchSources();
                    }

                    resetLastModifiedEventRefs();
                  }
                }, 1000);
              }
              break;

            case EventTypes.ADDED:
              const created = sourcesCreated + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesCreated', created);

              // If not waiting, or we're at 100%, then proceed
              if (!isAwaitingInstrumentation || (isAwaitingInstrumentation && created >= sourcesToCreate)) {
                addNotification({ type: StatusType.Success, title: EventTypes.ADDED, message: `Successfully created ${created} sources` });
                setInstrumentAwait(false);
                setInstrumentCount('sourcesToCreate', 0);
                setInstrumentCount('sourcesCreated', 0);
                fetchSources();
              }
              break;

            case EventTypes.DELETED:
              const deleted = sourcesDeleted + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesDeleted', deleted);

              // If not waiting, or we're at 100%, then proceed
              if (!isAwaitingInstrumentation || (isAwaitingInstrumentation && deleted >= sourcesToDelete)) {
                addNotification({ type: StatusType.Success, title: EventTypes.DELETED, message: `Successfully deleted ${deleted} sources` });
                setInstrumentAwait(false);
                setInstrumentCount('sourcesToDelete', 0);
                setInstrumentCount('sourcesDeleted', 0);
              }
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
      resetLastModifiedEventRefs();
    };
  }, []);
};
