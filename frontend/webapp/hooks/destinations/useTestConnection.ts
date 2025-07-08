import { useMutation } from '@apollo/client';
import { TEST_CONNECTION_MUTATION } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, type DestinationFormData } from '@odigos/ui-kit/types';

interface TestConnectionResponse {
  succeeded: boolean;
  statusCode: number;
  destinationType: string;
  message: string;
  reason: string;
}

export const useTestConnection = () => {
  const { addNotification } = useNotificationStore();

  const notifyUser = (type: StatusType, title: string, message: string, hideFromHistory?: boolean) => {
    addNotification({ type, title, message, hideFromHistory });
  };

  // TODO: change mutation, to lazy query
  const [testConnectionMutation, { loading: isTestConnectionLoading, data }] = useMutation<{ testConnectionForDestination: TestConnectionResponse }, { destination: DestinationFormData }>(
    TEST_CONNECTION_MUTATION,
    {
      onError: (error) => notifyUser(StatusType.Error, error.name || Crud.Read, error.cause?.message || error.message),
    },
  );

  const testConnection = async (destination: DestinationFormData) => {
    const { data } = await testConnectionMutation({ variables: { destination: { ...destination, fields: destination.fields.map((f) => ({ ...f, value: f.value || '' })) } } });

    return data?.testConnectionForDestination;
  };

  return {
    testConnection,
    isTestConnectionLoading,
    testConnectionResult: data?.testConnectionForDestination,
  };
};
