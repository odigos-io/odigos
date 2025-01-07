import { useQuery } from '@apollo/client';
import { GET_METRICS } from '@/graphql/mutations/metrics';
import type { OverviewMetricsResponse } from '@/types';

export const useMetrics = () => {
  // TODO: don't fetch until we have sources and/or destinations
  const { data } = useQuery<OverviewMetricsResponse>(GET_METRICS, {
    skip: false,
    pollInterval: 3000,
  });

  return { metrics: data };
};
