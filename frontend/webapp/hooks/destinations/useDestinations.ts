import { QUERIES } from '@/utils/constants';
import { useQuery } from 'react-query';
import { getDestinationsTypes } from '@/services';

export function useDestinations() {
  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  function getCurrentDestinationByType(type: string) {
    for (let category of data.categories) {
      for (let item of category.items) {
        if (item.type === type) {
          return item;
        }
      }
    }
    return null;
  }

  return { getCurrentDestinationByType, destinationsTypes: data, isLoading };
}
