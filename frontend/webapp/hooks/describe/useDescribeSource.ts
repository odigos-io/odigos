import { useQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import type { DescribeSource, WorkloadId } from '@/types';

export const useDescribeSource = ({ namespace, name, kind }: WorkloadId) => {
  const { data, loading, error } = useQuery<DescribeSource>(DESCRIBE_SOURCE, {
    fetchPolicy: 'cache-first',
    variables: { namespace, name, kind },
  });

  return {
    data: data?.describeSource,
    loading,
    error,
  };
};
