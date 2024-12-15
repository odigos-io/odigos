import { useQuery } from '@apollo/client';
import { GET_METRICS } from '@/graphql/mutations/metrics';
import type { OverviewMetricsResponse } from '@/types';

export const useMetrics = () => {
  const { data } = useQuery<OverviewMetricsResponse>(GET_METRICS, {
    pollInterval: 3000,
  });

  return { metrics: data };
};
