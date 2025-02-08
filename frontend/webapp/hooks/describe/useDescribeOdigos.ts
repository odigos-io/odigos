import { useQuery } from '@apollo/client';
import { DESCRIBE_ODIGOS } from '@/graphql';
import type { DescribeOdigos } from '@odigos/ui-containers';

export const useDescribeOdigos = () => {
  const { data, loading, error } = useQuery<{ describeOdigos: DescribeOdigos }>(DESCRIBE_ODIGOS, {
    pollInterval: 5000,
  });

  const isPro = ['onprem', 'enterprise'].includes(data?.describeOdigos.tier.value || '');

  return {
    loading,
    error,
    data: data?.describeOdigos,
    isPro,
  };
};
