import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { usePaginatedStore, useStatusStore } from '@/store';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { type NotifyPayload, useNotificationStore, usePendingStore } from '@odigos/ui-containers';
import { CRD_TYPES, DISPLAY_TITLES, ENTITY_TYPES, getIdFromSseTarget, NOTIFICATION_TYPE, type WorkloadId } from '@odigos/ui-utils';

const CONNECTED = 'CONNECTED';

const EVENT_TYPES = {
  ADDED: 'Added',
  MODIFIED: 'Modified',
  DELETED: 'Deleted',
};

export const useSSE = () => {
  const { setPaginated } = usePaginatedStore();
  const { setPendingItems } = usePendingStore();
  const { title, setStatusStore } = useStatusStore();
  const { addNotification } = useNotificationStore();
  const { fetchDestinations } = useDestinationCRUD();
  const { fetchSources, fetchSourceById } = useSourceCRUD();

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

        if (notification.title !== EVENT_TYPES.MODIFIED && notification.crdType !== CONNECTED) {
          // SSE toast notification (for all events except "modified" and "connected")
          addNotification(notification);
        }

        // Handle specific CRD types
        if ([CONNECTED].includes(notification.crdType as string)) {
          // If the current status in store is API Token related, we don't want to override it with the connected message
          if (title !== DISPLAY_TITLES.API_TOKEN) {
            setStatusStore({ status: NOTIFICATION_TYPE.SUCCESS, title: notification.title as string, message: notification.message as string });
          }
        } else if ([CRD_TYPES.INSTRUMENTATION_CONFIG].includes(notification.crdType as CRD_TYPES)) {
          if (notification.title === EVENT_TYPES.MODIFIED && !!notification.target) {
            fetchSourceById(getIdFromSseTarget(notification.target, ENTITY_TYPES.SOURCE) as WorkloadId);
          } else {
            if (notification.title === EVENT_TYPES.DELETED) setPaginated(ENTITY_TYPES.SOURCE, []);
            fetchSources();
          }
        } else if ([CRD_TYPES.DESTINATION].includes(notification.crdType as CRD_TYPES)) {
          fetchDestinations();
        } else console.warn('Unhandled SSE for CRD type:', notification.crdType);

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
