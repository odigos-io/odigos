import { useConfig } from '../config';
import { useMutation } from '@apollo/client';
import { type DestinationInput } from '@/types';
import { TEST_CONNECTION_MUTATION } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-containers';
import { DISPLAY_TITLES, FORM_ALERTS, NOTIFICATION_TYPE } from '@odigos/ui-utils';

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

  const [testConnectionMutation, { loading, error, data }] = useMutation<{ testConnectionForDestination: TestConnectionResponse }, { destination: DestinationInput }>(TEST_CONNECTION_MUTATION, {
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

    testConnection: (destination: DestinationInput) => {
      if (config?.readonly) {
        addNotification({ type: NOTIFICATION_TYPE.WARNING, title: DISPLAY_TITLES.READONLY, message: FORM_ALERTS.READONLY_WARNING, hideFromHistory: true });
      } else {
        testConnectionMutation({ variables: { destination: { ...destination, fields: destination.fields.map((f) => ({ ...f, value: f.value || '' })) } } });
      }
    },
  };
};
