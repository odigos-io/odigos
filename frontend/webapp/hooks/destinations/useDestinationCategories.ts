import { useLazyQuery } from '@apollo/client';
import { GET_DESTINATION_CATEGORIES } from '@/graphql';
import type { GetDestinationCategoriesResult } from '@odigos/ui-kit/types';

export const useDestinationCategories = () => {
  const [queryCategories] = useLazyQuery<GetDestinationCategoriesResult>(GET_DESTINATION_CATEGORIES);

  const getDestinationCategories = async (): Promise<GetDestinationCategoriesResult | undefined> => {
    const { data } = await queryCategories();
    return data;
  };

  return {
    getDestinationCategories,
  };
};
