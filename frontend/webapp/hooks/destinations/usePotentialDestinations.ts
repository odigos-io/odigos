import { useMemo } from 'react';
import { safeJsonParse } from '@/utils';
import { useQuery } from '@apollo/client';
import { IAppState, useAppStore } from '@/store';
import { GetDestinationTypesResponse } from '@/types';
import { GET_DESTINATION_TYPE, GET_POTENTIAL_DESTINATIONS } from '@/graphql';

interface PotentialDestination {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: PotentialDestination[];
}

const checkIfConfigured = (configuredDest: IAppState['configuredDestinations'][0], potentialDest: PotentialDestination, autoFilledFields: Record<string, any>) => {
  const typesMatch = configuredDest.stored.type === potentialDest.type;
  if (!typesMatch) return false;

  let fieldsMatch = false;

  for (const { key, value } of configuredDest.form.fields) {
    if (Object.hasOwn(autoFilledFields, key)) {
      if (autoFilledFields[key] === value) {
        fieldsMatch = true;
      } else {
        fieldsMatch = false;
        break;
      }
    }
  }

  return fieldsMatch;
};

export const usePotentialDestinations = () => {
  const { configuredDestinations } = useAppStore();
  const { data: { destinationTypes } = {} } = useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);
  const { loading, error, data: { potentialDestinations } = {} } = useQuery<GetPotentialDestinationsData>(GET_POTENTIAL_DESTINATIONS);

  const mappedPotentialDestinations = useMemo(() => {
    if (!destinationTypes || !potentialDestinations) return [];

    // Create a deep copy of destination types to manipulate
    const categories: GetDestinationTypesResponse['destinationTypes']['categories'] = JSON.parse(JSON.stringify(destinationTypes.categories));

    // Map over the potential destinations
    return potentialDestinations
      .map((pd) => {
        for (const category of categories) {
          const autoFilledFields = safeJsonParse<{ [key: string]: string }>(pd.fields, {});
          const alreadyConfigured = !!configuredDestinations.find((cd) => checkIfConfigured(cd, pd, autoFilledFields));

          if (!alreadyConfigured) {
            const idx = category.items.findIndex((item) => item.type === pd.type);

            if (idx !== -1) {
              return {
                // Spread the matched destination type data into the potential destination
                ...category.items[idx],
                fields: autoFilledFields,
              };
            }
          }
        }

        return null;
      })
      .filter((pd) => !!pd);
  }, [configuredDestinations, destinationTypes, potentialDestinations]);

  return {
    loading,
    error,
    data: mappedPotentialDestinations,
  };
};
