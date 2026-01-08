import { useMemo } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { GET_TOKENS, UPDATE_API_TOKEN } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { Crud, StatusType, type TokenPayload } from '@odigos/ui-kit/types';

export const useTokenCRUD = () => {
  const { isReadonly } = useConfig();
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const { refetch, data, loading } = useQuery<{ computePlatform?: { apiTokens?: TokenPayload[] } }>(GET_TOKENS, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });

  const [mutateUpdate] = useMutation<{ updateApiToken: boolean }>(UPDATE_API_TOKEN, {
    onError: (error) => {
      notifyUser(StatusType.Error, error.name || Crud.Update, error.cause?.message || error.message);
    },
    onCompleted: () => {
      notifyUser(StatusType.Success, Crud.Update, 'API Token updated');
      refetch();
    },
  });

  const updateToken = async (token: string) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      await mutateUpdate({ variables: { token } });
    }
  };

  const tokens = useMemo(() => data?.computePlatform?.apiTokens || [], [data?.computePlatform?.apiTokens?.length]);

  return {
    loading,
    tokens,
    updateToken,
  };
};
