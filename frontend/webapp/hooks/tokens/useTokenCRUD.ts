import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { UPDATE_API_TOKEN } from '@/graphql/mutations';
import { useComputePlatform } from '../compute-platform';
import { useNotificationStore } from '@odigos/ui-containers';
import { ACTION, DISPLAY_TITLES, FORM_ALERTS } from '@/utils';
import { ENTITY_TYPES, getSseTargetFromId, NOTIFICATION_TYPE } from '@odigos/ui-utils';

interface UseTokenCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useTokenCRUD = (params?: UseTokenCrudParams) => {
  const { data: config } = useConfig();
  const { data, refetch } = useComputePlatform();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, ENTITY_TYPES.ACTION) : undefined,
      hideFromHistory,
    });
  };

  const handleError = (actionType: string, message: string) => {
    notifyUser(NOTIFICATION_TYPE.ERROR, actionType, message);
    params?.onError?.(actionType);
  };

  const handleComplete = (actionType: string, message: string, id?: string) => {
    notifyUser(NOTIFICATION_TYPE.SUCCESS, actionType, message, id);
    refetch();
    params?.onSuccess?.(actionType);
  };

  const [updateToken, uState] = useMutation<{ updateApiToken: boolean }>(UPDATE_API_TOKEN, {
    onError: (error) => handleError(error.name || ACTION.UPDATE, error.cause?.message || error.message),
    onCompleted: () => handleComplete(ACTION.UPDATE, 'API Token updated'),
  });

  return {
    loading: uState.loading,
    tokens: data?.computePlatform?.apiTokens || [],

    updateToken: async (token: string) => {
      if (config?.readonly) {
        notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, undefined, true);
      } else {
        await updateToken({ variables: { token } });
      }
    },
  };
};
