import { useNotify } from '../useNotify';
import type { ActionInput } from '@/types';
import { useMutation } from '@apollo/client';
import { CREATE_ACTION } from '@/graphql/mutations/action';
import { useComputePlatform } from '../compute-platform';

export const useCreateAction = ({ onSuccess }: { onSuccess?: () => void }) => {
  const [createAction, { loading }] = useMutation(CREATE_ACTION, {
    onError: (error) =>
      notify({
        message: error.message,
        title: 'Create Action',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      }),
    onCompleted: () => {
      refetch();
      onSuccess && onSuccess();
      notify({
        message: 'Successfully created',
        title: 'Create Action',
        type: 'success',
        target: 'notification',
        crdType: 'notification',
      });
    },
  });

  const { refetch } = useComputePlatform();
  const notify = useNotify();

  return {
    createAction: (action: ActionInput) => createAction({ variables: { action } }),
    loading,
  };
};
