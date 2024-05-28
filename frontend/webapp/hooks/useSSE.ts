import { addNotification, store } from '@/store';
import { useEffect, useRef } from 'react';

const URL = 'http://localhost:8085/events';

export function useSSE() {
  const eventBuffer = useRef({});

  useEffect(() => {
    const eventSource = new EventSource(URL);

    eventSource.onmessage = function (event) {
      const data = JSON.parse(event.data);
      const key = event.data;
      console.log({ key, data });

      // Check if the event is already in the buffer
      if (
        eventBuffer.current[key] &&
        eventBuffer.current[key].id > Date.now() - 2000
      ) {
        eventBuffer.current[key] = {
          id: Date.now(),
          title: data.event,
          type: data.type,
          target: data.target,
          crdType: data.crdType,
          messages: [data.data],
        };
        return;
      } else {
        // Add a new event to the buffer
        eventBuffer.current[key] = {
          id: Date.now(),
          title: data.event,
          type: data.type,
          target: data.target,
          crdType: data.crdType,
          messages: [data.data],
        };
      }

      // Combine messages for the notification
      const combinedMessage = eventBuffer.current[key].messages;

      // Dispatch the notification to the store
      store.dispatch(
        addNotification({
          id: eventBuffer.current[key].id,
          message: combinedMessage,
          title: eventBuffer.current[key].title,
          type: eventBuffer.current[key].type,
          target: eventBuffer.current[key].target,
          crdType: eventBuffer.current[key].crdType,
        })
      );
    };

    eventSource.onerror = function (event) {
      console.error('EventSource failed:', event);
    };

    // Clean up event source on component unmount
    return () => {
      eventSource.close();
    };
  }, []);
}
