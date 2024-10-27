import { useNotify } from '../useNotify';
import type { ActionInput } from '@/types';
import { useEffect, useState } from 'react';
import { useMutation } from '@apollo/client';
import { CREATE_ACTION } from '@/graphql/mutations/action';
import { useComputePlatform } from '../compute-platform';

export const useCreateAction = () => {
  const [done, setDone] = useState(false);
  const [createAction, { data, loading, error }] = useMutation(CREATE_ACTION);

  const { refetch } = useComputePlatform();
  const notify = useNotify();

  useEffect(() => {
    if (error) {
      notify({
        message: error.message,
        title: 'Create Action Error',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      });
    }
  }, [error]);

  useEffect(() => {
    if (data) {
      refetch();
      setDone(true);
    }
  }, [data]);

  return {
    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    loading,
    done,
  };
};
