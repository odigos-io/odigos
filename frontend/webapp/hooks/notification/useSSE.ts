import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useStatusStore } from '@/store';
import { useSourceCRUD } from '../sources';
import { StatusType } from '@odigos/ui-kit/types';
import { useDestinationCRUD } from '../destinations';
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

const DEBOUNCE_MS = 5000;

export const useSSE = () => {
  const { fetchSources } = useSourceCRUD();
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { setInstrumentAwait, setInstrumentCount } = useInstrumentStore();

  const clearStatusMessage = () => {
    const { priorityMessage, setStatusStore } = useStatusStore.getState();
    if (!priorityMessage) setStatusStore({ status: StatusType.Default, message: '', leftIcon: undefined });
  };

  const maxRetries = 10;
  const retryCount = useRef(0);

  const eventRef = useRef<{
    interval: NodeJS.Timeout | null;
    timestamp: number | null;
  } | null>(null);

  const resetEventInterval = () => {
    if (eventRef.current) {
      if (eventRef.current.interval) {
        clearInterval(eventRef.current.interval);
        eventRef.current.interval = null;
      }
      eventRef.current.timestamp = null;
    }
  };

  const handleEventInterval = (successCallback: () => void) => {
    if (!eventRef.current) {
      eventRef.current = {
        interval: null,
        timestamp: null,
      };
    }

    if (eventRef.current.interval) clearInterval(eventRef.current.interval);
    eventRef.current.timestamp = Date.now();

    // if last message was over `XXX` seconds ago, fetch all sources...
    // the interval runs a timestamp check every 1 second - once the condition is met, the interval is cleared.
    eventRef.current.interval = setInterval(() => {
      const timeSinceLastModified = Date.now() - (eventRef.current?.timestamp || 0);

      if (timeSinceLastModified > DEBOUNCE_MS) {
        resetEventInterval();
        successCallback();
      }
    }, 1000);
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
          if (!isAwaitingInstrumentation) setInstrumentAwait(true);

          switch (notification.title) {
            case EventTypes.ADDED:
              const { sourcesToCreate, sourcesCreated } = useInstrumentStore.getState();
              if (sourcesToCreate > 0) {
                const totalCreated = sourcesCreated + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
                setInstrumentCount('sourcesCreated', totalCreated);
              }

              handleEventInterval(() => {
                addNotification({ type: StatusType.Success, title: EventTypes.ADDED, message: notification.message || 'Successfully created sources' });

                fetchSources();
                clearStatusMessage();

                setInstrumentAwait(false);
                setInstrumentCount('sourcesToCreate', 0);
                setInstrumentCount('sourcesCreated', 0);
              });
              break;

            case EventTypes.MODIFIED:
              handleEventInterval(() => {
                fetchSources();
                clearStatusMessage();
              });
              break;

            case EventTypes.DELETED:
              const { sourcesToDelete, sourcesDeleted } = useInstrumentStore.getState();
              if (sourcesToDelete > 0) {
                const totalDeleted = sourcesDeleted + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
                setInstrumentCount('sourcesDeleted', totalDeleted);
              }

              handleEventInterval(() => {
                addNotification({ type: StatusType.Success, title: EventTypes.DELETED, message: notification.message || 'Successfully deleted sources' });

                fetchSources();
                clearStatusMessage();

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
      resetEventInterval();
      clearStatusMessage();
    };
  }, []);
};
