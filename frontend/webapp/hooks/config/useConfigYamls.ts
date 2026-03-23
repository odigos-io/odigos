import { useQuery } from '@apollo/client';
import { GET_CONFIG_YAMLS } from '@/graphql';
import { type FetchedConfigYamls } from '@odigos/ui-kit/types';

export const useConfigYamls = () => {
  const { data } = useQuery<FetchedConfigYamls>(GET_CONFIG_YAMLS);

  return {
    configYamls: data?.configYamls?.configs || [],
  };
};
