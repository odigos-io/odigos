import { useNotify } from '../useNotify';
import type { ActionInput } from '@/types';
import { useMutation } from '@apollo/client';
import { UPDATE_ACTION } from '@/graphql/mutations/action';
import { useComputePlatform } from '../compute-platform';
import { useDrawerStore } from '@/store';

export const useUpdateAction = () => {
  const [updateAction, { loading }] = useMutation(UPDATE_ACTION, {
    onError: (error) => {
      notify({
        message: error.message,
        title: 'Update Action',
        type: 'error',
        target: 'notification',
        crdType: 'notification',
      });
    },
    onCompleted: () => {
      setDrawerItem(null);
      refetch();
      notify({
        message: 'Successfully updated',
        title: 'Update Action',
        type: 'success',
        target: 'notification',
        crdType: 'notification',
      });
    },
  });

  const { setSelectedItem: setDrawerItem } = useDrawerStore((store) => store);
  const { refetch } = useComputePlatform();
  const notify = useNotify();

  return {
    updateAction: (id: string, action: ActionInput) => updateAction({ variables: { id, action } }),
    loading,
  };
};
