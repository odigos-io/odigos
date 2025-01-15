import { useEffect } from 'react';
import { useTokenCRUD } from '.';
import { useTimeAgo } from '../common';
import { NOTIFICATION_TYPE } from '@/types';
import { isOverTime, SEVEN_DAYS_IN_MS } from '@/utils';
import { useConnectionStore, useNotificationStore } from '@/store';

// This hook is responsible for tracking the tokens and their expiration times.
// When a token is about to expire or has expired, a notification is added to the notification store, and the connection status is updated accordingly.

export const useTokenTracker = () => {
  const timeago = useTimeAgo();
  const { tokens } = useTokenCRUD();
  const { setTokenStatus } = useConnectionStore();
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    tokens.forEach(({ expiresAt, name }) => {
      if (isOverTime(expiresAt)) {
        const notif = {
          type: NOTIFICATION_TYPE.WARNING,
          title: 'API Token',
          message: `The token "${name}" has expired ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setTokenStatus({
          tokenExpired: true,
          tokenExpiring: false,
          title: notif.title,
          message: notif.message,
        });
      } else if (isOverTime(expiresAt, SEVEN_DAYS_IN_MS)) {
        const notif = {
          type: NOTIFICATION_TYPE.WARNING,
          title: 'API Token',
          message: `The token "${name}" is about to expire ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setTokenStatus({
          tokenExpired: false,
          tokenExpiring: true,
          title: notif.title,
          message: notif.message,
        });
      }
    });
  }, [tokens]);

  return {};
};
