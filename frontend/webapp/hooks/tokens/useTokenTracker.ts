import { useEffect } from 'react';
import { useTokenCRUD } from '.';
import { useStatusStore } from '@/store';
import { useTimeAgo } from '@odigos/ui-kit/hooks';
import { isOverTime } from '@odigos/ui-kit/functions';
import { StatusType } from '@odigos/ui-kit/types';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { DISPLAY_TITLES, TOKEN_ABOUT_TO_EXPIRE } from '@odigos/ui-kit/constants';

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
          type: StatusType.Error,
          title: DISPLAY_TITLES.API_TOKEN,
          message: `The token "${name}" has expired ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore({ status: notif.type, title: notif.title, message: notif.message });
      } else if (isOverTime(expiresAt, TOKEN_ABOUT_TO_EXPIRE)) {
        const notif = {
          type: StatusType.Warning,
          title: DISPLAY_TITLES.API_TOKEN,
          message: `The token "${name}" is about to expire ${timeago.format(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore({ status: notif.type, title: notif.title, message: notif.message });
      }
    });
  }, [tokens]);

  return {};
};
