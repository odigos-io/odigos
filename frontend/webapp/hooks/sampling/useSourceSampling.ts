import { useCallback } from 'react';
import { useLazyQuery } from '@apollo/client';
import { Crud, StatusType, type WorkloadId } from '@odigos/ui-kit/types';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { GET_SOURCE_SAMPLING } from '@/graphql';
import type { SourceSampling, SourceSamplingQueryResponse } from '@/types';

interface UseSourceSampling {
  fetchSourceSampling: (source: WorkloadId) => Promise<SourceSampling | undefined>;
}

// Wraps the `sourceSampling(workloadId)` query — returns the cluster-wide
// sampling rules pre-filtered per container of the given source (workload),
// based on each rule's source scope. Use from the Source Drawer "Sampling"
// tab to render only the rules that actually apply to the source.
//
// Example: await fetchSourceSampling({ namespace: 'demo', kind: 'Deployment', name: 'membership' })
//       => { workloadId: { ... }, containers: [{ containerName: 'membership', noisyOperations: [...], ... }] }
export const useSourceSampling = (): UseSourceSampling => {
  const { addNotification } = useNotificationStore();

  const [queryFn] = useLazyQuery<SourceSamplingQueryResponse, { workloadId: WorkloadId }>(GET_SOURCE_SAMPLING, {
    fetchPolicy: 'network-only',
  });

  const fetchSourceSampling: UseSourceSampling['fetchSourceSampling'] = useCallback(
    async (source) => {
      const { error, data } = await queryFn({ variables: { workloadId: source } });
      if (error) {
        addNotification({ type: StatusType.Error, title: Crud.Read, message: error.cause?.message || error.message });
        return undefined;
      }
      return data?.sourceSampling;
    },
    [queryFn, addNotification],
  );

  return { fetchSourceSampling };
};
