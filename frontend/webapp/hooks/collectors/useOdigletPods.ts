import { useQuery } from '@apollo/client';
import { GET_ODIGLET_PODS_WITH_METRICS } from '@/graphql/queries/collectors';

export type CollectorPodMetrics = {
  metricsAcceptedRps: number;
  metricsDroppedRps: number;
  exporterSuccessRps: number;
  exporterFailedRps: number;
  window: string;
  lastScrape?: string | null;
};

export type OdigletPodInfo = {
  name: string;
  namespace: string;
  ready: string;
  status?: string | null;
  restartsCount: number;
  nodeName: string;
  creationTimestamp: string;
  image: string;
  collectorMetrics?: CollectorPodMetrics | null;
};

type QueryResult = {
  odigletPods: OdigletPodInfo[];
};

export function useOdigletPods() {
  const { data, loading, error, refetch } = useQuery<QueryResult>(GET_ODIGLET_PODS_WITH_METRICS, {
    pollInterval: 5000,
  });

  return {
    pods: data?.odigletPods ?? [],
    loading,
    error,
    refetch,
  };
}


