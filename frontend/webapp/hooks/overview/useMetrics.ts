import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { useDestinationCRUD } from '../destinations';
import type { OverviewMetricsResponse } from '@/types';
import { GET_METRICS } from '@/graphql/mutations/metrics';

const data: OverviewMetricsResponse = {
  getOverviewMetrics: {
    sources: [
      {
        namespace: 'default',
        kind: 'Deployment',
        name: 'coupon',
        totalDataSent: 0,
        throughput: 1100,
      },
      {
        namespace: 'default',
        kind: 'Deployment',
        name: 'frontend',
        totalDataSent: 0,
        throughput: 10000,
      },
      {
        namespace: 'default',
        kind: 'Deployment',
        name: 'inventory',
        totalDataSent: 0,
        throughput: 1100,
      },
      {
        namespace: 'default',
        kind: 'Deployment',
        name: 'membership',
        totalDataSent: 0,
        throughput: 100,
      },
      {
        namespace: 'default',
        kind: 'Deployment',
        name: 'pricing',
        totalDataSent: 0,
        throughput: 100,
      },
    ],
    destinations: [
      {
        id: 'odigos.io.dest.jaeger-6gffq',
        totalDataSent: 0,
        throughput: 30000,
      },
    ],
  },
};

export const useMetrics = () => {
  const { sources } = useSourceCRUD();
  const { destinations } = useDestinationCRUD();

  // const { data } = useQuery<OverviewMetricsResponse>(GET_METRICS, {
  //   skip: !sources.length && !destinations.length,
  //   pollInterval: 3000,
  // });

  return { metrics: data };
};
