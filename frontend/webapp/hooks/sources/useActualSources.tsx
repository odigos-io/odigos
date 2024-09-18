import { useComputePlatform } from '../compute-platform';

export function useActualSources() {
  const { data } = useComputePlatform();

  return {
    sources: data?.computePlatform.k8sActualSources || [],
  };
}
