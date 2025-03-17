import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { GET_TOKENS, UPDATE_API_TOKEN } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { CRUD, NOTIFICATION_TYPE, TokenPayload } from '@odigos/ui-kit/types';

export const useTokenCRUD = () => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: NOTIFICATION_TYPE, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const { refetch, data, loading } = useQuery<{ computePlatform?: { apiTokens?: TokenPayload[] } }>(GET_TOKENS, {
    onError: (error) => notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  const [mutateUpdate] = useMutation<{ updateApiToken: boolean }>(UPDATE_API_TOKEN, {
    onError: (error) => {
      notifyUser(NOTIFICATION_TYPE.ERROR, error.name || CRUD.UPDATE, error.cause?.message || error.message);
    },
    onCompleted: () => {
      notifyUser(NOTIFICATION_TYPE.SUCCESS, CRUD.UPDATE, 'API Token updated');
      refetch();
    },
  });

  const updateToken = async (token: string) => {
    if (isReadonly) {
      notifyUser(NOTIFICATION_TYPE.WARNING, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutateUpdate({ variables: { token } });
    }
  };

  return {
    loading,
    tokens: data?.computePlatform?.apiTokens || [],
    updateToken,
  };
};
