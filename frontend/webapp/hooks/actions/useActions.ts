import { QUERIES } from '@/utils';
import { useQuery } from 'react-query';
import { getActions } from '@/services';

export function useActions() {
  const { isLoading, data } = useQuery([QUERIES.API_ACTIONS], getActions);

  return {
    isLoading,
    actions: data || [],
  };
}
