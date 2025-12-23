import { GET_METRICS } from '@/graphql';
import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import type { Metrics } from '@odigos/ui-kit/types';
import { useDestinationCRUD } from '../destinations';

export const useMetrics = () => {
  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  const { data } = useQuery<{ getOverviewMetrics: Metrics }>(GET_METRICS, {
    skip: !sources.length && !destinations.length,
    pollInterval: 10000,
  });

  return {
    metrics: data?.getOverviewMetrics || {
      sources: [],
      destinations: [],
    },
  };
};
