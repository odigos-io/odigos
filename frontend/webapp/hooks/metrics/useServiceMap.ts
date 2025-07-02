import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { GET_SERVICE_MAP } from '@/graphql';
import type { ServiceMapSources } from '@odigos/ui-kit/types';

export const useServiceMap = () => {
  const { sources } = useSourceCRUD();

  const { data } = useQuery<{ getServiceMap: { services: ServiceMapSources } }>(GET_SERVICE_MAP, {
    skip: !sources.length,
    pollInterval: 3000,
  });

  return {
    serviceMap: data?.getServiceMap?.services || [],
  };
};
