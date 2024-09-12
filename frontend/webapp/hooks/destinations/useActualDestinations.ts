import { GET_DESTINATIONS } from '@/graphql';
import { Destination } from '@/types';
import { useQuery } from '@apollo/client';

export const useActualDestination = () => {
  const { loading, error, data } = useQuery<{ destinations: Destination[] }>(
    GET_DESTINATIONS
  );

  return {
    loading,
    error,
    destinations: data?.destinations || [],
  };
};
