import { useNotificationStore } from '@/store';
import { Notification } from '@/types';

export type NotifyPayload = Omit<Notification, 'id' | 'dismissed' | 'seen' | 'time'>;

export const useNotify = () => {
  const { addNotification } = useNotificationStore();

  const notify = ({ type, title, message, crdType, target }: NotifyPayload) => {
    addNotification({ type, title, message, crdType, target });
  };

  return notify;
};
