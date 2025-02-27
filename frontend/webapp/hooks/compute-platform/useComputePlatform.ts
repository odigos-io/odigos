import { useQuery } from '@apollo/client';
import { type ComputePlatform } from '@/@types';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { CRUD, NOTIFICATION_TYPE } from '@odigos/ui-utils';
import { useNotificationStore } from '@odigos/ui-containers';

export const useComputePlatform = () => {
  const { addNotification } = useNotificationStore();

  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM, {
    fetchPolicy: 'cache-and-network',
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
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
