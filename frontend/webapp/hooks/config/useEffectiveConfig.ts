import { useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { GET_EFFECTIVE_CONFIG } from '@/graphql';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { Crud, type EffectiveConfig, StatusType } from '@odigos/ui-kit/types';

interface FetchedEffectiveConfig {
  effectiveConfig?: EffectiveConfig;
}

export const useEffectiveConfig = () => {
  const { data, loading, error, refetch } = useQuery<FetchedEffectiveConfig>(GET_EFFECTIVE_CONFIG);
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
    effectiveConfig: data?.effectiveConfig || null,
    effectiveConfigLoading: loading,
    refetchEffectiveConfig: refetch,
  };
};
