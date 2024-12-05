import { DestinationInput } from '@/types';
import { useMutation } from '@apollo/client';
import { TEST_CONNECTION_MUTATION } from '@/graphql';

interface TestConnectionResponse {
  succeeded: boolean;
  statusCode: number;
  destinationType: string;
  message: string;
  reason: string;
}

export const useTestConnection = () => {
  const [testConnectionMutation, { loading, error, data }] = useMutation<{ testConnectionForDestination: TestConnectionResponse }, { destination: DestinationInput }>(TEST_CONNECTION_MUTATION, {
    onError: (error, clientOptions) => {
      console.error('Error testing connection:', error);
    },
    onCompleted: (data, clientOptions) => {
      console.log('Successfully tested connection:', data);
    },
  });

  return {
    testConnection: (destination: DestinationInput) => testConnectionMutation({ variables: { destination } }),
    loading,
    error,
    data,
  };
};
