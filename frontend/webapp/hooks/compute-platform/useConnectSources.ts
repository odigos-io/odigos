// hooks/useConnectSources.ts
import { useMutation } from '@apollo/client';
import { useSelector, useDispatch } from 'react-redux';
import { CREATE_SOURCE } from '@/graphql';
import { IAppState, setSources } from '@/store';
import { K8sActualSource } from '@/types';
import { useComputePlatform } from './useComputePlatform';

type UseConnectSourcesHook = {
  createSources: (
    namespace: string,
    sources: K8sActualSource[]
  ) => Promise<void>;
  loading: boolean;
  error?: Error;
};

export const useConnectSources = (): UseConnectSourcesHook => {
  const [persistSourcesMutation, { loading, error }] =
    useMutation(CREATE_SOURCE);
  const dispatch = useDispatch();
  const sources = useSelector(({ app }: { app: IAppState }) => app.sources);
  const { data } = useComputePlatform();

  const createSources = async (namespace: string) => {
    try {
      let formattedSources = [];
      Object.keys(sources).forEach((key) => {
        const newSources = {
          sources: sources[key].map((source) => ({
            name: source.name,
            kind: source.kind,
            selected: source.selected,
          })),
        };

        formattedSources.push(newSources);
      });
      await persistSourcesMutation({
        variables: { namespace, sources: formattedSources },
      });
    } catch (e) {
      console.error('Error creating sources:', e);
      throw e;
    }
  };

  return {
    createSources,
    loading,
    error,
  };
};
