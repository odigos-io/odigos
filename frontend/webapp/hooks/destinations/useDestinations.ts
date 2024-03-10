import { QUERIES } from '@/utils/constants';
import { useQuery } from 'react-query';
import { Destination, DestinationsSortType } from '@/types';
import { useEffect, useState } from 'react';
import { getDestinations, getDestinationsTypes } from '@/services';

export function useDestinations() {
  const { isLoading, data, isError, error } = useQuery(
    [QUERIES.API_DESTINATION_TYPES],
    getDestinationsTypes
  );

  const [sortedDestinations, setSortedDestinations] = useState<
    Destination[] | undefined
  >(undefined);

  const {
    isLoading: destinationLoading,
    data: destinationList,
    refetch: refetchDestinations,
  } = useQuery<Destination[]>([QUERIES.API_DESTINATIONS], getDestinations);

  useEffect(() => {
    setSortedDestinations(destinationList || []);
  }, [destinationList]);

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

  function sortDestinations(condition: string) {
    const sorted = [...(destinationList || [])].sort((a, b) => {
      switch (condition) {
        case DestinationsSortType.TYPE:
          return a.type.localeCompare(b.type);
        case DestinationsSortType.NAME:
          const nameA = a.name || '';
          const nameB = b.name || '';
          return nameA.localeCompare(nameB);

        default:
          return 0;
      }
    });

    setSortedDestinations(sorted);
  }

  function filterDestinationsBySignal(signals: string[]) {
    const filteredData = destinationList?.filter((action) => {
      return signals.some((signal) => action.signals[signal]);
    });

    setSortedDestinations(filteredData);
  }

  return {
    isLoading,
    destinationLoading,
    destinationsTypes: data,
    destinationList: sortedDestinations || [],
    sortDestinations,
    refetchDestinations,
    filterDestinationsBySignal,
    getCurrentDestinationByType,
  };
}
