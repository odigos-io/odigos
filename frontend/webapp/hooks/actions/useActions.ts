import { QUERIES } from '@/utils';
import { useQuery } from 'react-query';
import { getActions } from '@/services';

export function useActions() {
  const { isLoading, data } = useQuery([QUERIES.API_ACTIONS], getActions);

  function getActionById(id: string) {
    return data?.find((action) => action.id === id);
  }

  return {
    isLoading,
    actions: data || [],
    getActionById,
  };
}
