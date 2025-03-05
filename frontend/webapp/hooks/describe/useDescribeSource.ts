import { useLazyQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-containers';
import { CRUD, NOTIFICATION_TYPE, type DescribeSource, type WorkloadId } from '@odigos/ui-utils';

export const useDescribeSource = () => {
  const { addNotification } = useNotificationStore();

  const [fetchDescribeSource, { data, loading, error }] = useLazyQuery<{ describeSource: DescribeSource }, WorkloadId>(DESCRIBE_SOURCE, {
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
    data: data?.describeSource,
    fetchDescribeSource,
  };
};
