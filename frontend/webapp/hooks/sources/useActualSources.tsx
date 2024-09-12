import { GET_SOURCES } from '@/graphql';
import { ManagedSource } from '@/types';
import { useQuery } from '@apollo/client';

export function useActualSources() {
  const { loading, error, data } = useQuery<{ actualSources: ManagedSource[] }>(
    GET_SOURCES
  );

  return {
    loading,
    error,
    sources: data?.actualSources || [],
  };

  return {};
}
