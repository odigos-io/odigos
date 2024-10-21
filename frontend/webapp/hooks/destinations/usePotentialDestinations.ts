import { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { useQuery } from '@apollo/client';
import { GetDestinationTypesResponse } from '@/types';
import { GET_DESTINATION_TYPE, GET_POTENTIAL_DESTINATIONS } from '@/graphql';

interface DestinationDetails {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: DestinationDetails[];
}

export const usePotentialDestinations = () => {
  const { data: destinationTypesData } =
    useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);
  const { loading, error, data } = useQuery<GetPotentialDestinationsData>(
    GET_POTENTIAL_DESTINATIONS
  );

  const mappedPotentialDestinations = useMemo(() => {
    if (!destinationTypesData || !data) return [];

    // Create a deep copy of destination types to manipulate
    const destinationTypesCopy = JSON.parse(
      JSON.stringify(destinationTypesData.destinationTypes.categories)
    );

    // Map over the potential destinations
    return data.potentialDestinations.map((destination) => {
      for (const category of destinationTypesCopy) {
        const index = category.items.findIndex(
          (item) => item.type === destination.type
        );
        if (index !== -1) {
          // Spread the matched destination type data into the potential destination
          const matchedType = category.items[index];
          category.items.splice(index, 1); // Remove the matched item from destination types
          return {
            ...destination,
            ...matchedType,
            fields: safeJsonParse<{ [key: string]: string }>(
              destination.fields,
              {}
            ),
          };
        }
      }
      return destination;
    });
  }, [destinationTypesData, data]);

  return {
    loading,
    error,
    data: mappedPotentialDestinations,
  };
};
