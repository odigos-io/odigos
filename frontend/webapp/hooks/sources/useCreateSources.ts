import { CREATE_SOURCE } from '@/graphql';
import { K8sActualSource } from '@/types';
import { useMutation } from '@apollo/client';
import { useState } from 'react';

type CreateSourceResponse = {
  persistK8sSources: boolean;
};

type CreateSourceVariables = {
  namespace: string;
  sources: K8sActualSource[];
};

export const useCreateSource = () => {
  const [createSourceMutation, { data, loading, error }] = useMutation<
    CreateSourceResponse,
    CreateSourceVariables
  >(CREATE_SOURCE);

  const [success, setSuccess] = useState<boolean | null>(null);

  const createSource = async (
    namespace: string,
    sources: K8sActualSource[]
  ) => {
    try {
      const result = await createSourceMutation({
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
    createSource,
    success,
    loading,
    error,
  };
};
