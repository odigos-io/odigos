import { useEffect } from 'react';
import { useTokenCRUD } from '.';
import { useTimeAgo } from '@odigos/ui-kit/hooks';
import { StatusType } from '@odigos/ui-kit/types';
import { StatusKeys, useStatusStore } from '@/store';
import { isOverTime } from '@odigos/ui-kit/functions';
import { useNotificationStore } from '@odigos/ui-kit/store';
import { TOKEN_ABOUT_TO_EXPIRE } from '@odigos/ui-kit/constants';

// This hook is responsible for tracking the tokens and their expiration times.
// When a token is about to expire or has expired, a notification is added to the notification store, and the connection status is updated accordingly.

export const useTokenTracker = () => {
  const { formatTimeAgo } = useTimeAgo();
  const { tokens } = useTokenCRUD();
  const { setStatusStore } = useStatusStore();
  const { addNotification } = useNotificationStore();

  useEffect(() => {
    tokens.forEach(({ expiresAt, name }) => {
      if (isOverTime(expiresAt)) {
        const notif = {
          type: StatusType.Error,
          message: `The token "${name}" has expired ${formatTimeAgo(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore(StatusKeys.Token, { status: notif.type, label: notif.message });
      } else if (isOverTime(expiresAt, TOKEN_ABOUT_TO_EXPIRE)) {
        const notif = {
          type: StatusType.Warning,
          message: `The token "${name}" is about to expire ${formatTimeAgo(expiresAt)}.`,
        };

        addNotification(notif);
        setStatusStore(StatusKeys.Token, { status: notif.type, label: notif.message });
      }
    });
  }, [tokens]);

  return {};
};
