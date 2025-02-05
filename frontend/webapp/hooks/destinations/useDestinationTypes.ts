import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_DESTINATION_TYPE } from '@/graphql';
import { type FetchedDestinationTypes } from '@/types';

const CATEGORIES_DESCRIPTION = {
  managed: 'Effortless Monitoring with Scalable Performance Management',
  'self hosted': 'Full Control and Customization for Advanced Application Monitoring',
};

export interface UseDestinationTypesResponse {
  destinations: (FetchedDestinationTypes['destinationTypes']['categories'][0] & { description: string })[];
}

export const useDestinationTypes = (): UseDestinationTypesResponse => {
  const { data } = useQuery<FetchedDestinationTypes>(GET_DESTINATION_TYPE);

  // Map fetched data
  const mapped: UseDestinationTypesResponse['destinations'] = useMemo(() => {
    return (data?.destinationTypes?.categories || []).map((category) => {
      const description = CATEGORIES_DESCRIPTION[category.name as keyof typeof CATEGORIES_DESCRIPTION];

      return { ...category, description };
    });
  }, [data]);

  return { destinations: mapped };
};
