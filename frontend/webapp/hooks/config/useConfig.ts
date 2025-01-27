'use client';
import { useEffect } from 'react';
// import { ACTION } from '@/utils';
// import { GET_CONFIG } from '@/graphql';
// import { useNotificationStore } from '@/store';
// import { useSuspenseQuery } from '@apollo/client';
// import { NOTIFICATION_TYPE, type Config } from '@/types';

const data = { config: { installation: 'FINISHED', readonly: true } };
const error = undefined;

export const useConfig = () => {
  // const { addNotification } = useNotificationStore();

  // const { data, error } = useSuspenseQuery<Config>(GET_CONFIG, {
  //   skip: typeof window === 'undefined',
  // });

  useEffect(() => {
    if (error) {
      // addNotification({
      //   type: NOTIFICATION_TYPE.ERROR,
      //   title: error.name || ACTION.FETCH,
      //   message: error.cause?.message || error.message,
      // });
    }
  }, [error]);

  return { data: data?.config };
};
