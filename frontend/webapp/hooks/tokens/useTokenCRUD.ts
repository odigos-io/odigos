import { useMutation } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { ACTION, getSseTargetFromId } from '@/utils';
import { UPDATE_API_TOKEN } from '@/graphql/mutations';
import { useComputePlatform } from '../compute-platform';
import { NOTIFICATION_TYPE, OVERVIEW_ENTITY_TYPES } from '@/types';

interface UseTokenCrudParams {
  onSuccess?: (type: string) => void;
  onError?: (type: string) => void;
}

export const useTokenCRUD = (params?: UseTokenCrudParams) => {
  const { data, refetch } = useComputePlatform();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, id?: string, hideFromHistory?: boolean) => {
    addNotification({
      type,
      title,
      message,
      crdType: OVERVIEW_ENTITY_TYPES.ACTION,
      target: id ? getSseTargetFromId(id, OVERVIEW_ENTITY_TYPES.ACTION) : undefined,
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

    updateToken: async (token: string) => await updateToken({ variables: { token } }),
  };
};
