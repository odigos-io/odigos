import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_METRICS } from '@/graphql/mutations/metrics';
import type { OverviewMetricsResponse } from '@/types';

export function useMetrics() {
  const { data, refetch } = useQuery<OverviewMetricsResponse>(GET_METRICS);

  useEffect(() => {
    const interval = setInterval(async () => await refetch(), 3000);
    return () => clearInterval(interval);
  }, [refetch]);

  return { metrics: data };
}
