import { useEffect } from 'react';
import { useNotify } from '.';

const URL = 'http://localhost:8085/events';

export function useSSE() {
  const notify = useNotify();
  useEffect(() => {
    const eventSource = new EventSource(URL);

    eventSource.onmessage = function (event) {
      const data = JSON.parse(event.data);
      console.log({ data });
      notify({
        message: data.data,
        title: data.event,
        type: data.type,
        target: data.target,
        crdType: data.crdType,
      });
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
