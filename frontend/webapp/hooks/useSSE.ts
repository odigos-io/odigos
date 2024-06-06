import { useEffect, useRef } from 'react';
import { API } from '@/utils';
import { addNotification, store } from '@/store';

export function useSSE() {
  const eventBuffer = useRef({});

  useEffect(() => {
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
