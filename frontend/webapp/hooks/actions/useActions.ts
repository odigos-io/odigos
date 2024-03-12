import { useEffect, useState } from 'react';
import { QUERIES } from '@/utils';
import { useMutation, useQuery } from 'react-query';
import { getActions, putAction } from '@/services';
import { ActionData, ActionItem, ActionsSortType } from '@/types';

export function useActions() {
  const { isLoading, data, refetch } = useQuery<ActionData[]>(
    [QUERIES.API_ACTIONS],
    getActions
  );

  const [sortedActions, setSortedActions] = useState<ActionData[] | undefined>(
    undefined
  );

  const { mutateAsync: updateAction } = useMutation((body: ActionItem) =>
    putAction(body?.id, body, body.type)
  );

  useEffect(() => {
    console.log({ data });
    setSortedActions(data || []);
  }, [data]);

  async function getActionById(id: string) {
    let actions = data;
    if (!data) {
      const res = await refetch();
      actions = res.data;
    }
    return actions?.find((action) => action.id === id);
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

  function filterActionsBySignal(signals: string[]) {
    const filteredData = data?.filter((action) => {
      return signals.some((signal) =>
        action.spec.signals.includes(signal.toUpperCase())
      );
    });

    setSortedActions(filteredData);
  }

  async function toggleActionStatus(
    ids: string[],
    disabled: boolean
  ): Promise<boolean> {
    for (const id of ids) {
      const action = await getActionById(id);
      if (action && action.spec.disabled !== disabled) {
        const body = {
          id: action.id,
          ...action.spec,
          disabled,
          type: action.type,
        };
        try {
          await updateAction(body);
        } catch (error) {
          return Promise.reject(false);
        }
      }
    }
    setTimeout(async () => {
      const res = await refetch();
      setSortedActions(res.data || []);
    }, 1000);

    return Promise.resolve(true);
  }

  async function handleActionsRefresh() {
    const res = await refetch();
    setSortedActions(res.data || []);
  }

  return {
    isLoading,
    actions: sortedActions || [],
    sortActions,
    getActionById,
    filterActionsBySignal,
    toggleActionStatus,
    refetch: handleActionsRefresh,
  };
}
