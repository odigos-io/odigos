import { useEffect } from 'react';
import { GET_GROUP_NAMES } from '@/graphql';
import { useLazyQuery } from '@apollo/client';
import { Crud, StatusType } from '@odigos/ui-kit/types';
import { useNotificationStore } from '@odigos/ui-kit/store';

interface UseGroupsCrud {
  groupNames: string[];
  groupNamesLoading: boolean;
  fetchGroupNames: () => void;
}

export const useGroupsCRUD = (): UseGroupsCrud => {
  const { addNotification } = useNotificationStore();

  const [fetchGroupNamesQuery, { loading, data, called }] = useLazyQuery<{ groupNames?: string[] }>(GET_GROUP_NAMES);

  const fetchGroupNames = async () => {
    const { error } = await fetchGroupNamesQuery();

    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  };

  useEffect(() => {
    if (!called) fetchGroupNames();
  }, []);

  return {
    groupNames: data?.groupNames || [],
    groupNamesLoading: loading,
    fetchGroupNames,
  };
};
