import { useComputePlatform } from '../compute-platform';
import { ActualDestination } from '@/types';

// Function to map raw data to the ActualDestination interface
const mapToActualDestination = (data: any): ActualDestination => ({
  id: data.id,
  name: data.name,
  type: data.type,
  exportedSignals: data.exportedSignals,
  fields: data.fields,
  conditions: data.conditions,
  destinationType: {
    type: data.destinationType.type,
    displayName: data.destinationType.displayName,
    imageUrl: data.destinationType.imageUrl,
    supportedSignals: data.destinationType.supportedSignals,
  },
});

export const useActualDestination = () => {
  const { data } = useComputePlatform();

  // Use the mapToActualDestination function to transform raw data
  const destinations =
    data?.computePlatform.destinations.map((destination: any) =>
      mapToActualDestination(destination)
    ) || [];

  return {
    destinations,
  };
};
