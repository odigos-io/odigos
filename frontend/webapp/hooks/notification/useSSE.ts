import { useEffect, useRef } from 'react';
import { API, DISPLAY_TITLES, NOTIF_CRD_TYPES } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import { useDestinationCRUD } from '../destinations';
import { usePaginatedSources } from '../compute-platform';
import { type NotifyPayload, useNotificationStore, usePendingStore, useStatusStore } from '@/store';

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
        const crdType = data.crdType || '';
        const notification: NotifyPayload = {
          type: data.type,
          title: data.event || '',
          message: data.data || '',
          crdType,
          target: data.target,
        };

        // SSE toast notification
        if (crdType !== NOTIF_CRD_TYPES.CONNECTED) addNotification(notification);

        // Handle specific CRD types
        if ([NOTIF_CRD_TYPES.CONNECTED].includes(crdType)) {
          if (title !== DISPLAY_TITLES.API_TOKEN) {
            setStatusStore({ status: NOTIFICATION_TYPE.SUCCESS, title: notification.title as string, message: notification.message as string });
          }
        } else if ([NOTIF_CRD_TYPES.INSTRUMENTATION_CONFIG, NOTIF_CRD_TYPES.INSTRUMENTATION_INSTANCE].includes(crdType)) {
          fetchSources();
        } else if ([NOTIF_CRD_TYPES.DESTINATION].includes(crdType)) {
          refetchDestinations();
        } else console.warn('Unhandled SSE for CRD type:', crdType);

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

      es.onerror = (event) => {
        console.error('EventSource failed:', event);
        es.close();

        // Retry connection with exponential backoff if below max retries
        if (retryCount.current < maxRetries) {
          retryCount.current += 1;
          const retryTimeout = Math.min(10000, 1000 * Math.pow(2, retryCount.current));

          setStatusStore({
            status: NOTIFICATION_TYPE.WARNING,
            title: 'Disconnected',
            message: `Disconnected from the server. Retrying connection (${retryCount.current})`,
          });

          setTimeout(() => connect(), retryTimeout);
        } else {
          console.error('Max retries reached. Could not reconnect to EventSource.');

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

    const es = connect();

    // Clean up event source on component unmount
    return () => {
      es.close();
    };
  }, [title]);
};
