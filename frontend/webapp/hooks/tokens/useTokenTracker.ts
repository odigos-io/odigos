import { useEffect } from 'react';
import { useTokenCRUD } from '.';
import { DISPLAY_TITLES, SEVEN_DAYS_IN_MS } from '@/utils';
import { useNotificationStore, useStatusStore } from '@/store';
import { isOverTime, NOTIFICATION_TYPE, useTimeAgo } from '@odigos/ui-components';

// This hook is responsible for tracking the tokens and their expiration times.
// When a token is about to expire or has expired, a notification is added to the notification store, and the connection status is updated accordingly.

export const useTokenTracker = () => {
  const timeago = useTimeAgo();
  const { tokens } = useTokenCRUD();
  const { setStatusStore } = useStatusStore();
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    tokens.forEach(({ expiresAt, name }) => {
      if (isOverTime(expiresAt)) {
        const notif = {
          type: NOTIFICATION_TYPE.ERROR,
          title: DISPLAY_TITLES.API_TOKEN,
          message: `The token "${name}" has expired ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore({
          status: NOTIFICATION_TYPE.ERROR,
          title: notif.title,
          message: notif.message,
        });
      } else if (isOverTime(expiresAt, SEVEN_DAYS_IN_MS)) {
        const notif = {
          type: NOTIFICATION_TYPE.WARNING,
          title: DISPLAY_TITLES.API_TOKEN,
          message: `The token "${name}" is about to expire ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore({
          status: NOTIFICATION_TYPE.WARNING,
          title: notif.title,
          message: notif.message,
        });
      }
    });
  }, [tokens]);

  return {};
};
