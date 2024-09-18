import { DestinationInput } from '@/types';
import { CREATE_DESTINATION } from '@/graphql';
import { useMutation } from '@apollo/client';

export function useCreateDestination() {
  const [createDestination] = useMutation(CREATE_DESTINATION);

  async function createNewDestination(destination: DestinationInput) {
    try {
      const { data } = await createDestination({
        variables: { destination },
      });
      return data?.createNewDestination?.id;
    } catch (error) {
      console.error('Error creating new destination:', error);
      throw error;
    }
  }

  return { createNewDestination };
}
