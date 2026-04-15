import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_WORKLOADS } from '@/graphql';
import type { K8sResourceKind, Workload } from '@odigos/ui-kit/types';

interface WorkloadFilter {
  namespace?: string;
  kind?: K8sResourceKind;
  name?: string;
  markedForInstrumentation?: boolean;
}

export const useWorkloads = (filter?: WorkloadFilter) => {
  const { data, loading } = useQuery<{ workloads: Workload[] }>(GET_WORKLOADS, {
    variables: {
      filter,
    },
  });

  const workloads = useMemo(() => data?.workloads ?? [], [data]);

  return {
    workloads,
    loading,
  };
};
