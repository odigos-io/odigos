import { QUERIES } from '@/utils/constants';
import { useQuery } from 'react-query';
import { getDestinations, getDestinationsTypes } from '@/services';
import { Destination } from '@/types';

export function useDestinations() {
  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch: refetchDestinations,
  } = useQuery<Destination[]>([QUERIES.API_DESTINATIONS], getDestinations);

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
  console.log({ destinationList, data });
  return {
    getCurrentDestinationByType,
    destinationsTypes: data,
    isLoading,
    destinationList: destinationList || [],
    destinationLoading,
  };
}
