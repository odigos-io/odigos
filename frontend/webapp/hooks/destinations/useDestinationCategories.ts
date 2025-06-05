import { useQuery } from '@apollo/client';
import { GET_DESTINATION_CATEGORIES } from '@/graphql';
import type { FetchedDestinationCategories } from '@/types';

export const useDestinationCategories = () => {
  const { data } = useQuery<FetchedDestinationCategories>(GET_DESTINATION_CATEGORIES);

  return {
    categories: data?.destinationCategories?.categories || [],
  };
};
