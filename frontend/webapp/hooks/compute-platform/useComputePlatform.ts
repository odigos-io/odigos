import { ComputePlatform } from '@/types';
import { useQuery } from '@apollo/client';
import { GET_COMPUTE_PLATFORM } from '@/graphql';

type UseComputePlatformHook = {
  data?: ComputePlatform;
  loading: boolean;
  error?: Error;
  refetch: () => void;
};

export const useComputePlatform = (): UseComputePlatformHook => {
  const { data, loading, error, refetch } =
    useQuery<ComputePlatform>(GET_COMPUTE_PLATFORM);

  return { data, loading, error, refetch };
};
