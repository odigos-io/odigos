'use client';
import { useEffect } from 'react';
import { ACTION } from '@/utils';
import { GET_CONFIG } from '@/graphql';
import { type FetchedConfig } from '@/types';
import { useSuspenseQuery } from '@apollo/client';
import { NOTIFICATION_TYPE } from '@odigos/ui-utils';
import { useNotificationStore } from '@odigos/ui-containers';

export const useConfig = () => {
  const { addNotification } = useNotificationStore();

  const { data, error } = useSuspenseQuery<FetchedConfig>(GET_CONFIG, {
    skip: typeof window === 'undefined',
  });

  useEffect(() => {
    if (error) {
      addNotification({
        type: NOTIFICATION_TYPE.ERROR,
        title: error.name || ACTION.FETCH,
        message: error.cause?.message || error.message,
      });
    }
  }, [error]);

  return { data: data?.config };
};
