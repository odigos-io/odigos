export interface SingleSourceMetricsResponse {
  namespace: string;
  kind: string;
  name: string;
  totalDataSent: number;
  throughput: number;
}

export interface SingleDestinationMetricsResponse {
  id: string;
  totalDataSent: number;
  throughput: number;
}

export interface OverviewMetricsResponse {
  getOverviewMetrics: {
    sources: SingleSourceMetricsResponse[];
    destinations: SingleDestinationMetricsResponse[];
  };
}
