import { useNotificationStore } from '@/store';
import { Notification } from '@/types';

export const useNotify = () => {
  const { addNotification } = useNotificationStore();

  const notify = ({
    type,
    title,
    message,
    crdType,
    target,
  }: {
    type: Notification['type'];
    title?: Notification['title'];
    message?: Notification['message'];
    crdType?: Notification['crdType'];
    target?: Notification['target'];
  }) => {
    addNotification({ type, title, message, crdType, target });
  };

  return notify;
};
