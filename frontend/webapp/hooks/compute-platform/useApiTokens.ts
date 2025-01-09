import { ACTION } from '@/utils';
import { useQuery } from '@apollo/client';
import { GET_API_TOKENS } from '@/graphql';
import { useNotificationStore } from '@/store';
import { type GetApiTokens, NOTIFICATION_TYPE } from '@/types';

export const useApiTokens = () => {
  const { addNotification } = useNotificationStore();

  const { data } = useQuery<GetApiTokens>(GET_API_TOKENS, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  return {
    data: data?.getApiTokens,
  };
};
