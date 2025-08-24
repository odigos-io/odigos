import { useEffect, useState } from 'react';
import type { Trace } from '@/types';
import { GET_TRACES } from '@/graphql';
import { useLazyQuery, useQuery } from '@apollo/client';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { useEntityStore, useNotificationStore } from '@odigos/ui-kit/store';

interface UseTracesParams {
  serviceName?: string;
}

const DEFAULT_LIMIT = 100;
const DEFAULT_HOURS_AGO = 24;

export const useTraces = ({ serviceName }: UseTracesParams) => {
  const [traces, setTraces] = useState<Trace[]>([]);

  const { sources } = useEntityStore();
  const { addNotification } = useNotificationStore();

  const { data } = useQuery<{ getTraces: Trace[] }>(GET_TRACES, {
    variables: { serviceName, limit: DEFAULT_LIMIT, hoursAgo: DEFAULT_HOURS_AGO },
    skip: !serviceName,
    onError: (error) =>
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      }),
  });

  const [fetchTraces] = useLazyQuery<{ getTraces: Trace[] }>(GET_TRACES, {
    onError: (error) =>
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      }),
  });

  useEffect(() => {
    if (!serviceName && sources.length > 0) {
      // If there is no service name, we should fetch traces for all sources
      sources.forEach(({ serviceName, name }) => {
        fetchTraces({ variables: { serviceName: serviceName || name, limit: DEFAULT_LIMIT, hoursAgo: DEFAULT_HOURS_AGO } }).then(({ data }) => {
          setTraces((prev) => {
            const newTraces = data?.getTraces?.filter((currTrace) => currTrace.spans.length > 0 && !prev.find((prevTrace) => prevTrace.traceID === currTrace.traceID)) ?? [];
            const arr = [...prev, ...newTraces];

            return arr;
          });
        });
      });
    }
  }, [serviceName, sources]);

  return {
    traces: serviceName ? data?.getTraces ?? [] : traces,
  };
};
