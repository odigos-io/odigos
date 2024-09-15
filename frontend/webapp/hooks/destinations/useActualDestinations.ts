import { useComputePlatform } from '../compute-platform';

export const useActualDestination = () => {
  const { data } = useComputePlatform();

  return {
    destinations: data?.computePlatform.destinations || [],
  };
};
