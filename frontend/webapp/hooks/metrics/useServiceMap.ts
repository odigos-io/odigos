import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { GET_SERVICE_MAP } from '@/graphql';
import type { ServiceMapSources } from '@odigos/ui-kit/types';

export const useServiceMap = () => {
  const { sources } = useSourceCRUD();

  const { data, refetch } = useQuery<{ getServiceMap: { services: ServiceMapSources } }>(GET_SERVICE_MAP, {
    skip: !sources.length,
  });

  return {
    serviceMap: data?.getServiceMap?.services || [],
    refetch,
  };
};
