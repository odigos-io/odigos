import { useEffect, useState } from 'react';
import { useQuery } from '@apollo/client';
import { GET_METRICS } from '@/graphql/mutations/metrics';
import type { ActualDestination, K8sActualSource, OverviewMetricsResponse } from '@/types';

export function useMetrics({ sources, destinations }: { sources: K8sActualSource[]; destinations: ActualDestination[] }) {
  const { data, refetch } = useQuery<OverviewMetricsResponse>(GET_METRICS);

  // this is just to re-render the mockup data
  const [_, setCount] = useState(0);

  useEffect(() => {
    const interval = setInterval(async () => {
      await refetch();
      setCount((n) => n + 1);
    }, 1000);

    return () => {
      clearInterval(interval);
    };
  }, [refetch]);

  // this will 1st try to use real data, if non-exitant it will generate mockup data
  const metricsMockup: OverviewMetricsResponse = {
    sources: data?.sources?.length
      ? data.sources
      : sources.map(({ name, kind, namespace }) => ({
          namespace,
          kind,
          name,
          totalDataSent: 0,
          throughput: Math.random() * 50000,
        })),
    destinations: data?.destinations?.length
      ? data.destinations
      : destinations.map(({ id }) => ({
          id,
          totalDataSent: 0,
          throughput: Math.random() * 50000,
        })),
  };

  return { metrics: metricsMockup };
}
