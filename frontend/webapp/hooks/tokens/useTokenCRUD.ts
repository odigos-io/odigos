import { useMemo } from 'react';
import { useConfig } from '../config';
import { useMutation, useQuery } from '@apollo/client';
import { GET_TOKENS, UPDATE_API_TOKEN } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { Crud, StatusType, type TokenPayload, type UpdateTokenFunc } from '@odigos/ui-kit/types';

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
    onError: () => {},
    onCompleted: () => {
      notifyUser(StatusType.Success, Crud.Update, 'API Token updated');
      refetch();
    },
  });

  const updateToken: UpdateTokenFunc = async (token) => {
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      const { errors, data } = await mutateUpdate({ variables: { token } });

      return {
        // @ts-expect-error the returned type for apollo errors is in-fact not an array but a regular object
        error: errors?.message as string | undefined,
        data,
      };
    }

    return undefined;
  };

  const tokens = useMemo(() => data?.computePlatform?.apiTokens || [], [data?.computePlatform?.apiTokens]);

  return {
    loading,
    tokens,
    updateToken,
  };
};
