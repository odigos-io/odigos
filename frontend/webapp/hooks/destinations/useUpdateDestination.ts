// src/hooks/useUpdateDestination.ts

import { UPDATE_DESTINATION } from '@/graphql';
import { useDrawerStore } from '@/store';
import { DestinationInput } from '@/types';
import { useMutation } from '@apollo/client';
import { useComputePlatform } from '../compute-platform';

export function useUpdateDestination() {
  const [updateDestinationMutation] = useMutation(UPDATE_DESTINATION);

  const setDrawerItem = useDrawerStore(({ setSelectedItem }) => setSelectedItem);
  const { refetch } = useComputePlatform();

  async function updateExistingDestination(id: string, destination: DestinationInput) {
    try {
      const { data } = await updateDestinationMutation({
        variables: { id, destination },
      });

      setDrawerItem(null);
      refetch();

      return data?.updateDestination?.id;
    } catch (error) {
      console.error('Error updating destination:', error);
      throw error;
    }
  }

  return { updateExistingDestination };
}
