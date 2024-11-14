import { useEffect, useRef, useState } from 'react';
import { API } from '@/utils';
import { addNotification, store } from '@/store';

export function useSSE() {
  const eventBuffer = useRef({});
  const [retryCount, setRetryCount] = useState(0);
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
        if (
          eventBuffer.current[key] &&
          eventBuffer.current[key].id > Date.now() - 2000
        ) {
          eventBuffer.current[key] = notification;
          return;
        } else {
          // Add a new event to the buffer
          eventBuffer.current[key] = notification;
        }

        // Dispatch the notification to the store
        store.dispatch(
          addNotification({
            id: eventBuffer.current[key].id,
            message: eventBuffer.current[key].message,
            title: eventBuffer.current[key].title,
            type: eventBuffer.current[key].type,
            target: eventBuffer.current[key].target,
            crdType: eventBuffer.current[key].crdType,
          })
        );

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
            const retryTimeout = Math.min(
              10000,
              1000 * Math.pow(2, newRetryCount)
            );

            setTimeout(() => {
              connect();
            }, retryTimeout);

            return newRetryCount;
          } else {
            console.error(
              'Max retries reached. Could not reconnect to EventSource.'
            );
            store.dispatch(
              addNotification({
                id: Date.now().toString(),
                message: 'Could not reconnect to EventSource.',
                title: 'Error',
                type: 'error',
                target: 'notification',
                crdType: 'notification',
              })
            );
            return prevRetryCount;
          }
        });
      };

      return eventSource;
    };

    const eventSource = connect();

    // Clean up event source on component unmount
    return () => {
      eventSource.close();
    };
  }, []);
}
