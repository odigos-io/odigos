// sources.ts
import { useMutation } from '@apollo/client';
import { UPDATE_K8S_ACTUAL_SOURCE } from '@/graphql';
import { WorkloadId, PatchSourceRequestInput } from '@/types';

export function useUpdateSource() {
  const [updateSourceMutation, { data, loading, error }] = useMutation(
    UPDATE_K8S_ACTUAL_SOURCE
  );

  const updateSource = async (
    sourceId: WorkloadId,
    patchSourceRequest: PatchSourceRequestInput
  ) => {
    try {
      const response = await updateSourceMutation({
        variables: {
          sourceId,
          patchSourceRequest,
        },
      });
      return response;
    } catch (err) {
      console.error('Error updating source:', err);
      throw err;
    }
  };

  return {
    updateSource,
    data,
    loading,
    error,
  };
}
