import { safeJsonParse } from '@/utils';
import type { ActionItem } from '@/types';
import { useComputePlatform } from '../compute-platform';

// Define the hook
export const useGetActions = () => {
  const { data } = useComputePlatform();

  return {
    actions:
      data?.computePlatform?.actions?.map((item) => {
        const parsedSpec = typeof item.spec === 'string' ? safeJsonParse(item.spec, {} as ActionItem) : item.spec;

        return {
          ...item,
          spec: parsedSpec,
        };
      }) || [],
  };
};
