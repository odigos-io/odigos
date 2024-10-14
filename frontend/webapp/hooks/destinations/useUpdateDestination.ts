// src/hooks/useUpdateDestination.ts

import { UPDATE_DESTINATION } from '@/graphql';
import { useDrawerStore } from '@/store';
import { DestinationInput } from '@/types';
import { useMutation } from '@apollo/client';

export function useUpdateDestination() {
  const [updateDestinationMutation] = useMutation(UPDATE_DESTINATION);

  const setDrawerItem = useDrawerStore(
    ({ setSelectedItem }) => setSelectedItem
  );

  async function updateExistingDestination(
    id: string,
    destination: DestinationInput
  ) {
    try {
      const { data } = await updateDestinationMutation({
        variables: { id, destination },
      });
      setDrawerItem({ id, item: data?.updateDestination, type: 'destination' });
      console.log({ data });
      return data?.updateDestination?.id;
    } catch (error) {
      console.error('Error updating destination:', error);
      throw error;
    }
  }

  return { updateExistingDestination };
}
