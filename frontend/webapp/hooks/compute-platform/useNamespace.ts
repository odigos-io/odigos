import { useQuery } from '@apollo/client';
import { GET_NAMESPACES } from '@/graphql';
import { ComputePlatform, K8sActualNamespace } from '@/types';
import { useEffect } from 'react';

type UseNamespaceHook = {
  data?: K8sActualNamespace;
  loading: boolean;
  error?: Error;
};

export const useNamespace = (
  namespaceName: string | undefined
): UseNamespaceHook => {
  const { data, loading, error } = useQuery<ComputePlatform>(GET_NAMESPACES, {
    skip: !namespaceName,
    variables: { namespaceName },
  });

  useEffect(() => {
    console.log({ data });
    console.log({ error });
  }, [data, error]);

  return {
    data: data?.computePlatform.k8sActualNamespace,
    loading,
    error,
  };
};
