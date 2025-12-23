import { useLazyQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DESCRIBE_ODIGOS, DESCRIBE_SOURCE } from '@/graphql';
import { Crud, StatusType, type DescribeOdigos, type DescribeSource, type WorkloadId } from '@odigos/ui-kit/types';

export const useDescribe = () => {
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const [fetchDescribeOdigos] = useLazyQuery<{ describeOdigos: DescribeOdigos }>(DESCRIBE_ODIGOS, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });

  const [fetchDescribeSource] = useLazyQuery<{ describeSource: DescribeSource }, WorkloadId>(DESCRIBE_SOURCE, {
    onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
  });

  return {
    fetchDescribeOdigos,
    fetchDescribeSource: (payload: WorkloadId) => fetchDescribeSource({ variables: payload }),
  };
};
