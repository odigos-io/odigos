import { useLazyQuery } from '@apollo/client';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DESCRIBE_ODIGOS, DESCRIBE_SOURCE } from '@/graphql';
import { CRUD, STATUS_TYPE, type DescribeOdigos, type DescribeSource } from '@odigos/ui-kit/types';

export const useDescribe = () => {
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: STATUS_TYPE, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  const [fetchDescribeOdigos] = useLazyQuery<{ describeOdigos: DescribeOdigos }>(DESCRIBE_ODIGOS, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  const [fetchDescribeSource] = useLazyQuery<{ describeSource: DescribeSource }>(DESCRIBE_SOURCE, {
    onError: (error) => notifyUser(STATUS_TYPE.ERROR, error.name || CRUD.READ, error.cause?.message || error.message),
  });

  return {
    fetchDescribeOdigos,
    fetchDescribeSource,
  };
};
