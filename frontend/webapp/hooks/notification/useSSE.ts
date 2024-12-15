import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { NOTIFICATION_TYPE } from '@/types';
import { useComputePlatform } from '../compute-platform';
import { type NotifyPayload, useConnectionStore, useNotificationStore } from '@/store';

const modifyType = (notification: NotifyPayload) => {
  if (notification.title === 'Modified') {
    if (notification.message?.indexOf('ProcessTerminated') === 0 || notification.message?.indexOf('NoHeartbeat') === 0 || notification.message?.indexOf('Failed') === 0) {
      return NOTIFICATION_TYPE.ERROR;
    } else {
      return NOTIFICATION_TYPE.INFO;
    }
  }

  return notification.type;
};

export const useSSE = () => {
  const { addNotification } = useNotificationStore();
  const { setConnectionStore } = useConnectionStore();
  const { refetch: refetchComputePlatform } = useComputePlatform();

  const retryCount = useRef(0);
  const eventBuffer = useRef({});
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

        notification.type = modifyType(notification);

        // Check if the event is already in the buffer
        if (eventBuffer.current[key] && eventBuffer.current[key].id > Date.now() - 2000) {
          eventBuffer.current[key] = notification;
        } else {
          // Add a new event to the buffer
          eventBuffer.current[key] = notification;

          // Dispatch the notification to the store
          addNotification(notification);
          refetchComputePlatform();
        }

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

          setConnectionStore({
            connecting: false,
            active: false,
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

      setConnectionStore({
        connecting: false,
        active: true,
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
