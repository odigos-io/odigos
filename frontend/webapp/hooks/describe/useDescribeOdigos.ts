import { useLazyQuery } from '@apollo/client';
import { DESCRIBE_ODIGOS } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { CRUD, DescribeOdigos, NOTIFICATION_TYPE } from '@odigos/ui-kit/types';

export const useDescribeOdigos = () => {
  const { addNotification } = useNotificationStore();

  const [fetchDescribeOdigos, { data, loading, error }] = useLazyQuery<{ describeOdigos: DescribeOdigos }>(DESCRIBE_ODIGOS, {
    onError: (error) =>
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || CRUD.READ,
        message: error.cause?.message || error.message,
      }),
  });

  return {
    loading,
    error,
    data: data?.describeOdigos,
    fetchDescribeOdigos,
  };
};
