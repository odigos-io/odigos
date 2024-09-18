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

interface UseTestConnectionResult {
  testConnection: (
    destination: DestinationInput
  ) => Promise<TestConnectionResponse | undefined>;
  loading: boolean;
  error?: Error;
}

export const useTestConnection = (): UseTestConnectionResult => {
  const [testConnectionMutation, { loading, error }] = useMutation<
    { testConnectionForDestination: TestConnectionResponse },
    { destination: DestinationInput }
  >(TEST_CONNECTION_MUTATION);

  const testConnection = async (
    destination: DestinationInput
  ): Promise<TestConnectionResponse | undefined> => {
    try {
      const { data } = await testConnectionMutation({
        variables: { destination },
      });
      return data?.testConnectionForDestination;
    } catch (err) {
      console.error('Error testing connection:', err);
      return undefined;
    }
  };

  return { testConnection, loading, error };
};
