import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { StatusType } from '@odigos/ui-kit/types';
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

// SSE event counters for debugging
const sseEventCounts = {
  total: 0,
  byEvent: {} as Record<string, number>,
  byCrd: {} as Record<string, number>,
};

export const useSSE = () => {
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { fetchSources } = useSourceCRUD();

  const maxRetries = 10;
  const retryCount = useRef(0);

  const lastModifiedEventTimestamp = useRef<number | null>(null);
  const lastModifiedEventInterval = useRef<NodeJS.Timeout | null>(null);

  const resetLastModifiedEventRefs = () => {
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

        // Count SSE events
        sseEventCounts.total++;
        const eventKey = notification.title || 'unknown';
        const crdKey = notification.crdType || 'unknown';
        sseEventCounts.byEvent[eventKey] = (sseEventCounts.byEvent[eventKey] || 0) + 1;
        sseEventCounts.byCrd[crdKey] = (sseEventCounts.byCrd[crdKey] || 0) + 1;
        console.log(`[SSE] Event #${sseEventCounts.total}: ${eventKey} (${crdKey})`, {
          counts: { ...sseEventCounts },
          data: notification,
        });

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
              if (!isAwaitingInstrumentation) {
                if (lastModifiedEventInterval.current) clearInterval(lastModifiedEventInterval.current);
                lastModifiedEventTimestamp.current = Date.now();

                // Debounce: wait for MODIFIED_DEBOUNCE_MS after the last event before fetching all sources
                lastModifiedEventInterval.current = setInterval(async () => {
                  const timeSinceLastModified = Date.now() - (lastModifiedEventTimestamp.current || 0);

                  if (timeSinceLastModified > MODIFIED_DEBOUNCE_MS) {
                    await fetchSources();
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
