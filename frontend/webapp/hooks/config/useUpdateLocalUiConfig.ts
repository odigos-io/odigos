import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, type LocalUiConfigInput } from '@odigos/ui-kit/types';
import { UPDATE_LOCAL_UI_CONFIG, RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS } from '@/graphql/mutations';

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

  const [resetMutate, { loading: resetLoading }] = useMutation<{ resetLocalUiConfigToFactoryDefaults: boolean }>(RESET_LOCAL_UI_CONFIG_TO_FACTORY_DEFAULTS, {
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
        message: 'Local UI configuration reset to factory defaults',
      });
    },
  });

  const updateLocalUiConfig = (config: LocalUiConfigInput) => {
    return mutate({ variables: { config } });
  };

  const resetLocalUiConfigToFactoryDefaults = () => {
    return resetMutate();
  };

  return {
    updateLocalUiConfig,
    resetLocalUiConfigToFactoryDefaults,
    loading: loading || resetLoading,
  };
};
