import { useCallback } from 'react';
import { GET_K8S_MANIFEST } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { K8sResourceKind } from '@odigos/ui-kit/types';

interface UseK8sManifest {
  fetchK8sManifest: (namespace: string, kind: K8sResourceKind, name: string) => Promise<string | undefined>;
}

export const useK8sManifest = (): UseK8sManifest => {
  const [queryK8sManifest] = useLazyQuery<{ k8sManifest: string }>(GET_K8S_MANIFEST);

  const fetchK8sManifest = useCallback(
    async (namespace: string, kind: K8sResourceKind, name: string) => {
      const { data } = await queryK8sManifest({ variables: { namespace, kind, name } });
      return data?.k8sManifest;
    },
    [queryK8sManifest],
  );

  return {
    fetchK8sManifest,
  };
};
