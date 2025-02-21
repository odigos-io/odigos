import { GET_METRICS } from '@/graphql';
import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import { K8S_RESOURCE_KIND } from '@odigos/ui-utils';

export const useMetrics = () => {
  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  const { data } = useQuery<{
    getOverviewMetrics: {
      sources: {
        namespace: string;
        name: string;
        kind: K8S_RESOURCE_KIND;
        totalDataSent: number;
        throughput: number;
      }[];
      destinations: {
        id: string;
        totalDataSent: number;
        throughput: number;
      }[];
    };
  }>(GET_METRICS, {
    skip: !sources.length && !destinations.length,
    pollInterval: 5000,
  });

  return {
    metrics: data?.getOverviewMetrics || {
      sources: [],
      destinations: [],
    },
  };
};
