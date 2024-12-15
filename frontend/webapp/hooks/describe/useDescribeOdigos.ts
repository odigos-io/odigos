import { useQuery } from '@apollo/client';
import { DESCRIBE_ODIGOS } from '@/graphql';
import type { DescribeOdigos } from '@/types';

export const useDescribeOdigos = () => {
  const { data, loading, error } = useQuery<DescribeOdigos>(DESCRIBE_ODIGOS, {
    pollInterval: 5000,
  });

  return {
    data: data?.describeOdigos,
    loading,
    error,
  };
};
