import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import { useDestinationCRUD } from '../destinations';
import { usePaginatedSources } from '../compute-platform';
import { type NotifyPayload, useConnectionStore, useNotificationStore, usePendingStore } from '@/store';

export const useSSE = () => {
  const { setSseStatus } = useConnectionStore();
  const { setPendingItems } = usePendingStore();
  const { fetchSources } = usePaginatedSources();
  const { addNotification } = useNotificationStore();
  const { refetchDestinations } = useDestinationCRUD();

  const retryCount = useRef(0);
  const maxRetries = 10;

  useEffect(() => {
    const connect = () => {
      const eventSource = new EventSource(API.EVENTS);

      eventSource.onmessage = (event) => {
        const key = event.data;
        const data = JSON.parse(key);

        const notification: NotifyPayload = {
          type: data.type,
          title: data.event,
          message: data.data,
          crdType: data.crdType,
          target: data.target,
        };

        addNotification(notification);

        const crdType = notification.crdType || '';
        if (['InstrumentationConfig', 'InstrumentationInstance'].includes(crdType)) {
          fetchSources();
        } else if (['Destination'].includes(crdType)) {
          refetchDestinations();
        } else {
          console.warn('Unhandled SSE for CRD type:', crdType);
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

      eventSource.onerror = (event) => {
        console.error('EventSource failed:', event);
        eventSource.close();

        // Retry connection with exponential backoff if below max retries
        if (retryCount.current < maxRetries) {
          retryCount.current += 1;
          const retryTimeout = Math.min(10000, 1000 * Math.pow(2, retryCount.current));

          setTimeout(() => connect(), retryTimeout);
        } else {
          console.error('Max retries reached. Could not reconnect to EventSource.');

          setSseStatus({
            sseConnecting: false,
            sseStatus: NOTIFICATION_TYPE.ERROR,
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

      setSseStatus({
        sseConnecting: false,
        sseStatus: NOTIFICATION_TYPE.SUCCESS,
        title: 'Connection Alive',
        message: '',
      });

      return eventSource;
    };

    const eventSource = connect();

    // Clean up event source on component unmount
    return () => {
      eventSource.close();
    };
  }, []);
};
