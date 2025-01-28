import { useQuery } from '@apollo/client';
import { useEffect, useState } from 'react';
import { GET_DESTINATION_TYPE } from '@/graphql';
import { DestinationsCategory, GetDestinationTypesResponse } from '@/types';

const CATEGORIES_DESCRIPTION = {
  managed: 'Effortless Monitoring with Scalable Performance Management',
  'self hosted': 'Full Control and Customization for Advanced Application Monitoring',
};

export interface IDestinationListItem extends DestinationsCategory {
  description: string;
}

export function useDestinationTypes() {
  const [destinations, setDestinations] = useState<IDestinationListItem[]>([]);
  const { data } = useQuery<GetDestinationTypesResponse>(GET_DESTINATION_TYPE);

  useEffect(() => {
    if (data) {
      setDestinations(
        data.destinationTypes.categories.map((category) => ({
          name: category.name,
          description: CATEGORIES_DESCRIPTION[category.name as keyof typeof CATEGORIES_DESCRIPTION],
          items: category.items,
        })),
      );
    }
  }, [data]);

  return { destinations };
}
