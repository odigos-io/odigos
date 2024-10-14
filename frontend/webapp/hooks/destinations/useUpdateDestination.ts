// src/hooks/useUpdateDestination.ts

import { UPDATE_DESTINATION } from '@/graphql';
import { DestinationInput } from '@/types';
import { useMutation } from '@apollo/client';

export function useUpdateDestination() {
  const [updateDestinationMutation] = useMutation(UPDATE_DESTINATION);

  async function updateExistingDestination(
    id: string,
    destination: DestinationInput
  ) {
    try {
      const { data } = await updateDestinationMutation({
        variables: { id, destination },
      });
      return data?.updateDestination?.id;
    } catch (error) {
      console.error('Error updating destination:', error);
      throw error;
    }
  }

  return { updateExistingDestination };
}
