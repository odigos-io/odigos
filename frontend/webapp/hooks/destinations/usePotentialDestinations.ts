import { useMemo } from 'react';
import { useLazyQuery, useQuery } from '@apollo/client';
import { GET_POTENTIAL_DESTINATIONS } from '@/graphql';
import { deepClone, safeJsonParse } from '@odigos/ui-kit/functions';
import { useDestinationCategories } from './useDestinationCategories';
import { useSetupStore, type ISetupState } from '@odigos/ui-kit/store';
import type { GetPotentialDestinationsResult } from '@odigos/ui-kit/types';

interface PotentialDestination {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: PotentialDestination[];
}

export const usePotentialDestinations = () => {
  const [queryPotentialDests] = useLazyQuery<GetPotentialDestinationsResult>(GET_POTENTIAL_DESTINATIONS);

  const getPotentialDestinations = async (): Promise<GetPotentialDestinationsResult | undefined> => {
    const { data } = await queryPotentialDests();
    return data;
  };

  return {
    getPotentialDestinations,
  };
};
