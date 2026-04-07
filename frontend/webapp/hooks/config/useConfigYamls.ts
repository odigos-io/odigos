import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_CONFIG_YAMLS } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, StatusType, type ConfigYaml } from '@odigos/ui-kit/types';

interface FetchedConfigYamls {
  configYamls: ConfigYaml[];
}

export const useConfigYamls = () => {
  const { data, loading, error } = useQuery<FetchedConfigYamls>(GET_CONFIG_YAMLS);
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    if (error) {
      addNotification({
        type: StatusType.Error,
        title: error.name || Crud.Read,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  return {
    configYamls: data?.configYamls || [],
    configYamlsLoading: loading,
  };
};
