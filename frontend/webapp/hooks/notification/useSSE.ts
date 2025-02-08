import { useEffect, useRef } from 'react';
import { useStatusStore } from '@/store';
import { useDestinationCRUD } from '../destinations';
import { usePaginatedSources } from '../compute-platform';
import { API, SSE_CRD_TYPES, SSE_EVENT_TYPES } from '@/utils';
import { DISPLAY_TITLES, NOTIFICATION_TYPE } from '@odigos/ui-utils';
import { type NotifyPayload, useNotificationStore, usePendingStore } from '@odigos/ui-containers';

export const useSSE = () => {
  const { setPendingItems } = usePendingStore();
  const { fetchSources } = usePaginatedSources();
  const { title, setStatusStore } = useStatusStore();
  const { addNotification } = useNotificationStore();
  const { refetchDestinations } = useDestinationCRUD();

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

        if (notification.title !== SSE_EVENT_TYPES.MODIFIED && notification.crdType !== SSE_CRD_TYPES.CONNECTED) {
          // SSE toast notification (for all events except "modified" and "connected")
          addNotification(notification);
        }

        // Handle specific CRD types
        if ([SSE_CRD_TYPES.CONNECTED].includes(notification.crdType as string)) {
          if (title !== DISPLAY_TITLES.API_TOKEN) {
            setStatusStore({ status: NOTIFICATION_TYPE.SUCCESS, title: notification.title as string, message: notification.message as string });
          }
        } else if ([SSE_CRD_TYPES.INSTRUMENTATION_CONFIG, SSE_CRD_TYPES.INSTRUMENTATION_INSTANCE].includes(notification.crdType as string)) {
          fetchSources();
        } else if ([SSE_CRD_TYPES.DESTINATION].includes(notification.crdType as string)) {
          refetchDestinations();
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
