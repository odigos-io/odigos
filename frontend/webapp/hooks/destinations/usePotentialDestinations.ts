import { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { useQuery } from '@apollo/client';
import { GetDestinationTypesResponse } from '@/types';
import { GET_DESTINATION_TYPE, GET_POTENTIAL_DESTINATIONS } from '@/graphql';

interface PotentialDestination {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: PotentialDestination[];
}

export const usePotentialDestinations = () => {
  const { data: destinationTypesData } = useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);
  const { loading, error, data } = useQuery<GetPotentialDestinationsData>(GET_POTENTIAL_DESTINATIONS);

  const mappedPotentialDestinations = useMemo(() => {
    if (!destinationTypesData || !data) return [];

    // Create a deep copy of destination types to manipulate
    const categories: GetDestinationTypesResponse['destinationTypes']['categories'] = JSON.parse(JSON.stringify(destinationTypesData.destinationTypes.categories));

    // Map over the potential destinations
    return data.potentialDestinations
      .map((pd) => {
        for (const category of categories) {
          const index = category.items.findIndex((item) => item.type === pd.type);

          if (index !== -1) {
            // Spread the matched destination type data into the potential destination
            const matchedType = { ...category.items[index] };

            // Remove the matched item from destination types
            category.items.splice(index, 1);

            return {
              ...matchedType,
              fields: safeJsonParse<{ [key: string]: string }>(pd.fields, {}),
            };
          }
        }

        return null;
      })
      .filter((pd) => !!pd);
  }, [destinationTypesData, data]);

  return {
    loading,
    error,
    data: mappedPotentialDestinations,
  };
};
