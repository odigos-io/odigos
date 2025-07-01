import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { GET_SERVICE_MAP } from '@/graphql';

export const useServiceMap = () => {
  const { sources } = useSourceCRUD();

  // TODO: replace 'any' with the correct type after a release of UI-Kit: https://github.com/odigos-io/ui-kit/pull/207
  const { data } = useQuery<{ getServiceMap: { services: any[] } }>(GET_SERVICE_MAP, {
    skip: !sources.length,
    pollInterval: 3000,
  });

  return {
    serviceMap: data?.getServiceMap?.services || [],
  };
};
