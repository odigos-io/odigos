import { useComputePlatform } from '../compute-platform';

export const useGetInstrumentationRules = () => {
  const { data } = useComputePlatform();

  return {
    instrumentationRules: data?.computePlatform?.instrumentationRules || [],
  };
};
