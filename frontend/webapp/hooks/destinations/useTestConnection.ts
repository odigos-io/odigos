import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { TEST_CONNECTION_MUTATION } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { Crud, StatusType, type DestinationFormData } from '@odigos/ui-kit/types';

interface TestConnectionResponse {
  succeeded: boolean;
  statusCode: number;
  destinationType: string;
  message: string;
  reason: string;
}

export const useTestConnection = () => {
  const { isReadonly } = useConfig();
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
    if (isReadonly) {
      notifyUser(StatusType.Warning, DISPLAY_TITLES.READONLY, FORM_ALERTS.READONLY_WARNING, true);
    } else {
      const { data } = await testConnectionMutation({ variables: { destination: { ...destination, fields: destination.fields.map((f) => ({ ...f, value: f.value || '' })) } } });

      return data?.testConnectionForDestination;
    }
  };

  return {
    testConnection,
    isTestConnectionLoading,
    testConnectionResult: data?.testConnectionForDestination,
  };
};
