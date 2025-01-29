import { GET_METRICS } from '@/graphql';
import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import type { OverviewMetricsResponse } from '@/types';

export const useMetrics = () => {
  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  const { data } = useQuery<OverviewMetricsResponse>(GET_METRICS, {
    skip: !sources.length && !destinations.length,
    pollInterval: 3000,
  });

  return { metrics: data };
};
