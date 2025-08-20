import type { Trace } from '@/types';
import { GET_TRACES } from '@/graphql';
import { useQuery } from '@apollo/client';

interface UseTracesParams {
  serviceName: string;
}

export const useTraces = ({ serviceName }: UseTracesParams) => {
  const { data } = useQuery<{ getTraces: Trace[] }>(GET_TRACES, {
    variables: { serviceName },
    skip: !serviceName,
  });

  return {
    traces: data?.getTraces ?? [],
  };
};
