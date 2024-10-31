import { deriveTypeFromRule } from '@/utils/functions/derive-type-from-rule';
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
