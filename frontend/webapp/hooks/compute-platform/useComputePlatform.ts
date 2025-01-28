import { ACTION } from '@/utils';
import { useQuery } from '@apollo/client';
import { useNotificationStore } from '@/store';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { NOTIFICATION_TYPE, type ComputePlatform } from '@/types';

export const useComputePlatform = () => {
  const { addNotification } = useNotificationStore();

  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      }),
  });

  return {
    data,
    loading,
    error,
    refetch,
  };
};
