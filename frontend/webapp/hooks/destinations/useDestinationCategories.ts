import { useLazyQuery, useQuery } from '@apollo/client';
import { GET_DESTINATION_CATEGORIES } from '@/graphql';
import type { FetchedDestinationCategories } from '@/types';
import type { GetDestinationCategoriesResult } from '@odigos/ui-kit/types';

export const useDestinationCategories = () => {
  const { data } = useQuery<FetchedDestinationCategories>(GET_DESTINATION_CATEGORIES);
  const [queryCategories] = useLazyQuery<GetDestinationCategoriesResult>(GET_DESTINATION_CATEGORIES);

  const getDestinationCategories = async (): Promise<GetDestinationCategoriesResult | undefined> => {
    const { data } = await queryCategories();
    return data;
  };

  return {
    categories: data?.destinationCategories?.categories || [],
    getDestinationCategories,
  };
};
