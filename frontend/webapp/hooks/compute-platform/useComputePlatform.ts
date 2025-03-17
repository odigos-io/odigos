import { useQuery } from '@apollo/client';
import type { ComputePlatform } from '@/@types';
import { GET_COMPUTE_PLATFORM } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { CRUD, NOTIFICATION_TYPE } from '@odigos/ui-kit/types';

export const useComputePlatform = () => {
  const { addNotification } = useNotificationStore();

  const { data, loading, error, refetch } = useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM, {
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
