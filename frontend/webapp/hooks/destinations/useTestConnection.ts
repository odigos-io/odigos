import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { TEST_CONNECTION_MUTATION } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, FORM_ALERTS } from '@odigos/ui-kit/constants';
import { NOTIFICATION_TYPE, type DestinationFormData } from '@odigos/ui-kit/types';

interface TestConnectionResponse {
  succeeded: boolean;
  statusCode: number;
  destinationType: string;
  message: string;
  reason: string;
}

export const useTestConnection = () => {
  const { data: config } = useConfig();
  const { addNotification } = useNotificationStore();

  // TODO: change mutation, to lazy query
  const [testConnectionMutation, { loading, error, data }] = useMutation<{ testConnectionForDestination: TestConnectionResponse }, { destination: DestinationFormData }>(TEST_CONNECTION_MUTATION, {
    onError: (error) => {
      console.error('Error testing connection:', error);
    },
    onCompleted: (data) => {
      console.log('Successfully tested connection:', data);
    },
  });

  return {
    loading,
    error,
    data: data?.testConnectionForDestination,

    testConnection: (destination: DestinationFormData) => {
      if (config?.readonly) {
        addNotification({ type: NOTIFICATION_TYPE.WARNING, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
      } else {
        testConnectionMutation({ variables: { destination: { ...destination, fields: destination.fields.map((f) => ({ ...f, value: f.value || '' })) } } });
      }
    },
  };
};
