<<<<<<< HEAD
import { useMutation, useQuery } from '@apollo/client';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { ComputePlatform, PersistNamespaceItemInput } from '@/types';
import { NOTIFICATION } from '@/utils';
import { useNotify } from '../notification';

export const useNamespace = (namespaceName?: string, instrumentationLabeled = null as boolean | null) => {
  const notify = useNotify();
=======
import { NOTIFICATION } from '@/utils';
import { useNotify } from '../notification';
import { useMutation, useQuery } from '@apollo/client';
import { useComputePlatform } from './useComputePlatform';
import { GET_NAMESPACES, PERSIST_NAMESPACE } from '@/graphql';
import { ComputePlatform, PersistNamespaceItemInput } from '@/types';

export const useNamespace = (namespaceName?: string, instrumentationLabeled = null as boolean | null) => {
  const notify = useNotify();
  const cp = useComputePlatform();
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866

  const handleError = (title: string, message: string) => {
    notify({ type: NOTIFICATION.ERROR, title, message });
  };

  const handleComplete = (title: string, message: string) => {
    notify({ type: NOTIFICATION.SUCCESS, title, message });
  };

  const { data, loading, error } = useQuery<ComputePlatform>(GET_NAMESPACES, {
    skip: !namespaceName,
    fetchPolicy: 'cache-first',
    variables: { namespaceName, instrumentationLabeled },
  });

  const [persistNamespaceMutation] = useMutation(PERSIST_NAMESPACE, {
    onError: (error) => handleError('', error.message),
    onCompleted: (res, req) => {},
  });

  return {
<<<<<<< HEAD
=======
    allNamespaces: cp.data?.computePlatform.k8sActualNamespaces,
>>>>>>> a109419fc0a9639860b5769980d0020fce32e866
    persistNamespace: async (namespace: PersistNamespaceItemInput) => await persistNamespaceMutation({ variables: { namespace } }),
    data: data?.computePlatform.k8sActualNamespace,
    loading,
    error,
  };
};
