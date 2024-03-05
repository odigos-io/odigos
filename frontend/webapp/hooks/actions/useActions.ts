import { QUERIES } from '@/utils';
import { useQuery } from 'react-query';
import { getActions } from '@/services';
import { useEffect, useState } from 'react';
import { ActionData, ActionsSortType } from '@/types';

export function useActions() {
  const { isLoading, data } = useQuery<ActionData[]>(
    [QUERIES.API_ACTIONS],
    getActions
  );

  const [sortedActions, setSortedActions] = useState<ActionData[] | undefined>(
    undefined
  );

  useEffect(() => {
    setSortedActions(data || []);
  }, [data]);

  function getActionById(id: string) {
    return data?.find((action) => action.id === id);
  }

  function sortActions(condition: string) {
    const sorted = [...(data || [])].sort((a, b) => {
      switch (condition) {
        case ActionsSortType.TYPE:
          return a.type.localeCompare(b.type);
        case ActionsSortType.ACTION_NAME:
          // Assuming spec.actionName exists, otherwise sort them to the end.
          const nameA = a.spec?.actionName || '';
          const nameB = b.spec?.actionName || '';
          return nameA.localeCompare(nameB);
        case ActionsSortType.STATUS:
          // Treat missing 'disabled' as 'enabled'
          const statusA = a.spec?.disabled ? 1 : -1;
          const statusB = b.spec?.disabled ? 1 : -1;
          return statusA - statusB;
        default:
          return 0;
      }
    });

    setSortedActions(sorted);
  }

  return {
    isLoading,
    actions: sortedActions || [],
    sortActions,
    getActionById,
  };
}
