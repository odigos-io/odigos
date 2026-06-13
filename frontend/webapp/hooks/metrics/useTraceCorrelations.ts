import { useQuery } from '@apollo/client';
import { GET_TRACE_CORRELATIONS } from '@/graphql';

export type TraceCorrelationsAttribute = {
  key: string;
  value: string;
};

export type TraceCorrelationsOutputSeries = {
  attributes: TraceCorrelationsAttribute[];
  connectionCount: number;
  firstDetectedAt: string;
};

export type TraceCorrelationsInputGroup = {
  attributes: TraceCorrelationsAttribute[];
  outputs: TraceCorrelationsOutputSeries[];
};

export type TraceCorrelationsWorkload = {
  namespace: string;
  kind: string;
  name: string;
  containerName: string;
  inputs: TraceCorrelationsInputGroup[];
};

export type WorkloadFilter = {
  namespace?: string;
  kind?: string;
  name?: string;
};

type TraceCorrelationsResponse = {
  traceCorrelations: {
    workloads: TraceCorrelationsWorkload[];
  };
};

export const useTraceCorrelations = (filter?: WorkloadFilter) => {
  const { data, loading, error, refetch } = useQuery<TraceCorrelationsResponse>(GET_TRACE_CORRELATIONS, {
    variables: { filter: filter ?? null },
    pollInterval: 30_000,
  });

  return {
    workloads: data?.traceCorrelations?.workloads ?? [],
    loading,
    error,
    refetch,
  };
};
