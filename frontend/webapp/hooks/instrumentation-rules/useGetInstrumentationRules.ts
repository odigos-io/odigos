import { deriveTypeFromRule } from '@/utils';
import { useComputePlatform } from '../compute-platform';

export const useGetInstrumentationRules = () => {
  const { data } = useComputePlatform();

  return {
    instrumentationRules:
      data?.computePlatform?.instrumentationRules?.map((item) => {
        const type = deriveTypeFromRule(item);

        return {
          ...item,
          type,
        };
      }) || [],
  };
};
