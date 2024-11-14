import { useEffect, useRef, useState } from 'react';
import { useNotify } from './useNotify';
import { API, NOTIFICATION } from '@/utils';
import { useConnectionStore } from '@/store';

export function useSSE() {
  const notify = useNotify();
  const { setConnecting, setActive, setTitle, setMessage } = useConnectionStore();

  const [retryCount, setRetryCount] = useState(0);
  const eventBuffer = useRef({});
  const maxRetries = 10;

  useEffect(() => {
    const connect = () => {
      const eventSource = new EventSource(API.EVENTS);

      eventSource.onmessage = function (event) {
        const data = JSON.parse(event.data);
        const key = event.data;

        const notification = {
          id: Date.now(),
          message: data.data,
          title: data.event,
          type: data.type,
          target: data.target,
          crdType: data.crdType,
        };

        // Check if the event is already in the buffer
        if (eventBuffer.current[key] && eventBuffer.current[key].id > Date.now() - 2000) {
          eventBuffer.current[key] = notification;
          return;
        } else {
          // Add a new event to the buffer
          eventBuffer.current[key] = notification;
        }

        // Dispatch the notification to the store
        notify({
          type: eventBuffer.current[key].type,
          title: eventBuffer.current[key].title,
          message: eventBuffer.current[key].message,
          crdType: eventBuffer.current[key].crdType,
          target: eventBuffer.current[key].target,
        });

        // Reset retry count on successful connection
        setRetryCount(0);
      };

      eventSource.onerror = function (event) {
        console.error('EventSource failed:', event);
        eventSource.close();

        // Retry connection with exponential backoff if below max retries
        setRetryCount((prevRetryCount) => {
          if (prevRetryCount < maxRetries) {
            const newRetryCount = prevRetryCount + 1;
            const retryTimeout = Math.min(10000, 1000 * Math.pow(2, newRetryCount));

            setTimeout(() => connect(), retryTimeout);

            return newRetryCount;
          } else {
            console.error('Max retries reached. Could not reconnect to EventSource.');

            notify({
              type: NOTIFICATION.ERROR,
              title: 'Connection Error',
              message: 'Connection to the server failed. Please reboot the application.',
            });

            setConnecting(false);
            setActive(false);
            setTitle(`Connection lost on ${new Date().toLocaleString()}`);
            setMessage('Please reboot the application');

            return prevRetryCount;
          }
        });
      };

      setConnecting(false);
      setActive(true);
      setTitle('Connection Alive');
      setMessage('');

      return eventSource;
    };

    const eventSource = connect();

    // Clean up event source on component unmount
    return () => {
      eventSource.close();
    };
  }, []);
}
