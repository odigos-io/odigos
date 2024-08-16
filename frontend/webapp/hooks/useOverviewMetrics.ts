import { useQuery } from 'react-query';
import { getOverviewMetrics } from '@/services/metrics';

export function useOverviewMetrics() {
  const { data: metrics } = useQuery([], getOverviewMetrics, {
    refetchInterval: 5000,
  });

  return { metrics };
}
