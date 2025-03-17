import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { useStatusStore } from '@/store';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { DISPLAY_TITLES } from '@odigos/ui-kit/constants';
import { getIdFromSseTarget } from '@odigos/ui-kit/functions';
import { CRD_TYPES, ENTITY_TYPES, NOTIFICATION_TYPE, type WorkloadId } from '@odigos/ui-kit/types';
import { type NotifyPayload, useInstrumentStore, useNotificationStore, usePendingStore } from '@odigos/ui-kit/store';

const CONNECTED = 'CONNECTED';

const EVENT_TYPES = {
  ADDED: 'Added',
  MODIFIED: 'Modified',
  DELETED: 'Deleted',
};

export const useSSE = () => {
  const { setPendingItems } = usePendingStore();
  const { addNotification } = useNotificationStore();
  const { title, setStatusStore } = useStatusStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { fetchSourcesPaginated, fetchSourceById } = useSourceCRUD();

  const retryCount = useRef(0);
  const maxRetries = 10;

  useEffect(() => {
    const connect = () => {
      const es = new EventSource(API.EVENTS);

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

        const isConnected = [CONNECTED].includes(notification.crdType as string);
        const isSource = [CRD_TYPES.INSTRUMENTATION_CONFIG].includes(notification.crdType as CRD_TYPES);
        const isDestination = [CRD_TYPES.DESTINATION].includes(notification.crdType as CRD_TYPES);

        if (!isConnected && !(isSource && isAwaitingInstrumentation) && notification.title !== EVENT_TYPES.MODIFIED) {
          addNotification(notification);
        }

        // Handle specific CRD types
        if (isConnected) {
          // If the current status in store is API Token related, we don't want to override it with the connected message
          if (title !== DISPLAY_TITLES.API_TOKEN) {
            setStatusStore({ status: NOTIFICATION_TYPE.SUCCESS, title: notification.title as string, message: notification.message as string });
          }
        } else if (isSource) {
          switch (notification.title) {
            case EVENT_TYPES.MODIFIED:
              if (!isAwaitingInstrumentation && !!notification.target) {
                const id = getIdFromSseTarget(notification.target, ENTITY_TYPES.SOURCE);
                fetchSourceById(id as WorkloadId);
              }
              break;

            case EVENT_TYPES.ADDED:
              const created = sourcesCreated + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesCreated', created);

              if (!isAwaitingInstrumentation || (isAwaitingInstrumentation && created >= sourcesToCreate)) {
                addNotification({ type: NOTIFICATION_TYPE.SUCCESS, title: EVENT_TYPES.ADDED, message: `Successfully created ${created} sources` });
                setInstrumentAwait(false);
                fetchSourcesPaginated();
              }
              break;

            case EVENT_TYPES.DELETED:
              const deleted = sourcesDeleted + Number(notification.message?.toString().replace(/[^\d]/g, '') || 0);
              setInstrumentCount('sourcesDeleted', deleted);

              if (!isAwaitingInstrumentation || (isAwaitingInstrumentation && deleted >= sourcesToDelete)) {
                addNotification({ type: NOTIFICATION_TYPE.SUCCESS, title: EVENT_TYPES.DELETED, message: `Successfully deleted ${deleted} sources` });
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

        // This works for now,
        // but in the future we might have to change this to "removePendingItems",
        // and remove the specific pending items based on their entityType and entityId
        setPendingItems([]);

        // This works for now,
        // but in the future we might have to change this to "removePendingItems",
        // and remove the specific pending items based on their entityType and entityId
        setPendingItems([]);

        // Reset retry count on successful connection
        retryCount.current = 0;
      };

      es.onerror = () => {
        es.close();

        if (retryCount.current < maxRetries) {
          retryCount.current += 1;
          setStatusStore({
            status: NOTIFICATION_TYPE.WARNING,
            title: 'Disconnected',
            message: `Disconnected from the server. Retrying connection (${retryCount.current})`,
          });

          // Retry connection with exponential backoff if below max retries
          setTimeout(() => connect(), Math.min(10000, 1000 * Math.pow(2, retryCount.current)));
        } else {
          setStatusStore({
            status: NOTIFICATION_TYPE.ERROR,
            title: `Connection lost on ${new Date().toLocaleString()}`,
            message: 'Please reboot the application',
          });
          addNotification({
            type: NOTIFICATION_TYPE.ERROR,
            title: 'Connection Error',
            message: 'Connection to the server failed. Please reboot the application.',
          });
        }
      };

      return es;
    };

    // Initialize event source connection
    const es = connect();
    // Clean up event source on component unmount
    return () => es.close();
  }, [title]);
};
