import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { useSourceCRUD } from '../sources';
import { GET_SERVICE_MAP } from '@/graphql';
import type { ServiceMapSources } from '@odigos/ui-kit/types';

export const useServiceMap = () => {
  const { sources } = useSourceCRUD();

  const { data, refetch } = useQuery<{ getServiceMap: { services: ServiceMapSources } }>(GET_SERVICE_MAP, {
    skip: !sources.length,
    // pollInterval: 3000,
  });

  // !!! this is a temporay workaround until we update the UI-Kit.
  //     the issue is: refetch does a reset on service X/Y positions (even after user-drags).
  //     so this is to ensure we receive data once, and keep the X/Y positions.
  useEffect(() => {
    if (!data?.getServiceMap?.services?.length) {
      const interval = setInterval(() => refetch(), 3000);
      return () => clearInterval(interval);
    }
  }, [data?.getServiceMap?.services?.length]);

  return {
    serviceMap: data?.getServiceMap?.services || [],
  };
};
