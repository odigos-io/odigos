import { useMemo } from 'react';
import { useQuery } from '@apollo/client';
import { GET_POTENTIAL_DESTINATIONS } from '@/graphql';
import { safeJsonParse } from '@/utils';

interface DestinationDetails {
  type: string;
  fields: string;
}

interface GetPotentialDestinationsData {
  potentialDestinations: DestinationDetails[];
}

// Custom hook
export const usePotentialDestinations = () => {
  const { loading, error, data } = useQuery<GetPotentialDestinationsData>(
    GET_POTENTIAL_DESTINATIONS
  );

  // Memoize the transformed data to avoid unnecessary recalculations
  const transformedData = useMemo(() => {
    if (!data) return null;

    return data.potentialDestinations.map((destination) => ({
      ...destination,
      fields: safeJsonParse<Record<string, string>>(destination.fields, {}),
    }));
  }, [data]);

  return {
    loading,
    error,
    data: transformedData,
  };
};
