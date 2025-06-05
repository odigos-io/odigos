import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_POTENTIAL_DESTINATIONS } from '@/graphql';
import { deepClone, safeJsonParse } from '@odigos/ui-kit/functions';
import { useDestinationCategories } from './useDestinationCategories';
import { useSetupStore, type ISetupState } from '@odigos/ui-kit/store';

interface PotentialDestination {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: PotentialDestination[];
}

const checkIfConfigured = (configuredDest: ISetupState['configuredDestinations'][0], potentialDest: PotentialDestination, autoFilledFields: Record<string, any>) => {
  const typesMatch = configuredDest.type === potentialDest.type;
  if (!typesMatch) return false;

  let fieldsMatch = false;

  for (const { key, value } of configuredDest.form.fields) {
    if (Object.hasOwn(autoFilledFields, key)) {
      // An exact match is when all "ifs" result in true.
      // If one resulted with false, it is not an exact match and therefore not an "already configured destination".
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
  const { configuredDestinations } = useSetupStore();
  const { categories } = useDestinationCategories();
  const { loading, data: { potentialDestinations } = {} } = useQuery<GetPotentialDestinationsData>(GET_POTENTIAL_DESTINATIONS);

  const mappedPotentialDestinations = useMemo(() => {
    if (!categories || !potentialDestinations) return [];

    // Create a deep copy of destination types to manipulate
    const parsed = deepClone<typeof categories>(categories);

    // Map over the potential destinations
    return potentialDestinations
      .map((pd) => {
        for (const category of parsed) {
          const autoFilledFields = safeJsonParse<{ [key: string]: string }>(pd.fields, {});
          const alreadyConfigured = configuredDestinations.find((cd) => checkIfConfigured(cd, pd, autoFilledFields));

          if (!alreadyConfigured) {
            const idx = category.items.findIndex((item) => item.type === pd.type);

            if (idx !== -1) {
              return {
                // Spread the matched destination type data into the potential destination
                ...category.items[idx],
                fields: category.items[idx].fields.map((field) => ({
                  ...field,
                  initialValue: autoFilledFields[field.name],
                })),
              };
            }
          }
        }

        return null;
      })
      .filter((pd) => pd);
  }, [configuredDestinations, categories, potentialDestinations]);

  return {
    loading,
    potentialDestinations: mappedPotentialDestinations,
  };
};
