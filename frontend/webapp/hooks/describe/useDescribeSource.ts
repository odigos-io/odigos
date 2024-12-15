import { useQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import type { DescribeSource, WorkloadId } from '@/types';

export const useDescribeSource = ({ namespace, name, kind }: WorkloadId) => {
  const { data, loading, error } = useQuery<DescribeSource>(DESCRIBE_SOURCE, {
    variables: { namespace, name, kind },
    pollInterval: 5000,
  });

  return {
    data: data?.describeSource,
    loading,
    error,
  };
};
