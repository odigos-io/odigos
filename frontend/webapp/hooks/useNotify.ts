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
    title: Notification['title'];
    message: Notification['message'];
    crdType: Notification['crdType'];
    target: Notification['target'];
  }) => {
    const date = new Date();

    addNotification({
      id: date.getTime().toString(),
      type,
      title,
      message,
      crdType,
      target,
      isNew: true,
      seen: false,
      time: date.toISOString(),
    });
  };

  return notify;
};
