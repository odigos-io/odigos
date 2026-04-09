import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { UPDATE_LOCAL_UI_CONFIG } from '@/graphql/mutations';
import { Crud, StatusType, type LocalUiConfigInput } from '@odigos/ui-kit/types';

export const useUpdateLocalUiConfig = () => {
  const { addNotification } = useNotificationStore();

  const [mutate, { loading }] = useMutation<{ updateLocalUiConfig: boolean }, { config: LocalUiConfigInput }>(UPDATE_LOCAL_UI_CONFIG, {
    onError: (error) => {
      addNotification({
        type: StatusType.Error,
        title: Crud.Update,
        message: error.message,
      });
    },
    onCompleted: () => {
      addNotification({
        type: StatusType.Success,
        title: Crud.Update,
        message: 'Local UI configuration updated successfully',
      });
    },
  });

  const updateLocalUiConfig = (config: LocalUiConfigInput) => {
    return mutate({ variables: { config } });
  };

  return {
    updateLocalUiConfig,
    loading,
  };
};
