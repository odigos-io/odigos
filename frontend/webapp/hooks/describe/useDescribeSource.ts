import { useQuery } from '@apollo/client';
import { DESCRIBE_SOURCE } from '@/graphql';
import { type DescribeSource } from '@/types';
import { type WorkloadId } from '@odigos/ui-utils';

export const useDescribeSource = (params?: WorkloadId) => {
  const { namespace, name, kind } = params || {};

  // TODO: change query, to lazy query
  const { data, loading, error } = useQuery<{ describeSource: DescribeSource }>(DESCRIBE_SOURCE, {
    skip: !namespace || !name || !kind,
    variables: { namespace, name, kind },
    pollInterval: 5000,
  });

  return {
    loading,
    error,
    data: data?.describeSource,
  };
};
