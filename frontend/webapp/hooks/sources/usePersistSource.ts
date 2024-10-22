import { useState } from 'react';
import { PERSIST_SOURCE } from '@/graphql';
import { useMutation } from '@apollo/client';
import { PersistSourcesArray } from '@/types';

type PersistSourceResponse = {
  persistK8sSources: boolean;
};

type PersistSourceVariables = {
  namespace: string;
  sources: PersistSourcesArray[];
};

export const usePersistSource = () => {
  const [persistSourceMutation, { data, loading, error }] = useMutation<
    PersistSourceResponse,
    PersistSourceVariables
  >(PERSIST_SOURCE);

  const [success, setSuccess] = useState<boolean | null>(null);

  const persistSource = async (
    namespace: string,
    sources: PersistSourcesArray[]
  ) => {
    try {
      const result = await persistSourceMutation({
        variables: {
          namespace,
          sources,
        },
      });
      setSuccess(result.data?.persistK8sSources ?? false);
    } catch (err) {
      setSuccess(false);
    }
  };

  return {
    persistSource,
    success,
    loading,
    error,
  };
};
