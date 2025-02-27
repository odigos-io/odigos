import { useLazyQuery } from '@apollo/client';
import { DESCRIBE_ODIGOS } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-containers';
import { CRUD, NOTIFICATION_TYPE, type DescribeOdigos } from '@odigos/ui-utils';

export const useDescribeOdigos = () => {
  const { addNotification } = useNotificationStore();

  const [fetchDescribeOdigos, { data, loading, error }] = useLazyQuery<{ describeOdigos: DescribeOdigos }>(DESCRIBE_ODIGOS, {
    fetchPolicy: 'cache-and-network',
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
