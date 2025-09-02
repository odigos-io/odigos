import { useEffect, useState } from 'react';
import type { Trace } from '@/types';
import { GET_TRACES } from '@/graphql';
import { useLazyQuery, useQuery } from '@apollo/client';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';

interface UseTracesParams {
  serviceName?: string;
}

interface GetTracesVariables {
  serviceName: string;
  limit?: number;
  hoursAgo?: number;
}

const DEFAULT_LIMIT = 100;
const DEFAULT_HOURS_AGO = 24;

export const useTraces = ({ serviceName }: UseTracesParams) => {
  const [traces, setTraces] = useState<Trace[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  const { sources } = useEntityStore();
  const { addNotification } = useNotificationStore();

  const { data, loading } = useQuery<{ getTraces: Trace[] }, GetTracesVariables>(GET_TRACES, {
    variables: { serviceName: serviceName || '', limit: DEFAULT_LIMIT, hoursAgo: DEFAULT_HOURS_AGO },
    skip: !serviceName,
    onError: (error) =>
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      }),
  });

  const [fetchTraces] = useLazyQuery<{ getTraces: Trace[] }, GetTracesVariables>(GET_TRACES, {
    onError: (error) =>
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      }),
  });

  const fetchAllTraces = async () => {
    setIsLoading(true);

    for await (const source of sources) {
      const { error, data } = await fetchTraces({
        variables: {
          serviceName: source.serviceName || source.name,
          limit: DEFAULT_LIMIT,
          hoursAgo: DEFAULT_HOURS_AGO,
        },
      });

      if (error) {
        addNotification({
          type: StatusType.Error,
          title: error.name || Crud.Read,
          message: error.cause?.message || error.message,
        });
      } else if (data?.getTraces) {
        setTraces((prev) => {
          const newTraces = data?.getTraces?.filter((currTrace) => currTrace.spans.length > 0 && !prev.find((prevTrace) => prevTrace.traceID === currTrace.traceID)) ?? [];
          const arr = [...prev, ...newTraces];

          return arr;
        });
      }
    }

    setIsLoading(false);
  };

  useEffect(() => {
    // If there is no service name, we should fetch traces for all sources
    if (!serviceName && sources.length > 0 && traces.length === 0) fetchAllTraces();
  }, [serviceName, sources.length, traces.length]);

  return {
    traces: serviceName ? data?.getTraces ?? [] : traces,
    isLoading: (isLoading || loading) && !traces.length,
  };
};
