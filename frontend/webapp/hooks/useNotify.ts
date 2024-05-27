import { addNotification, store } from '@/store';

export const useNotify = () => {
  const dispatch = store.dispatch;

  const notify = ({
    message,
    title,
    type,
    target,
    crdType,
  }: {
    message: string;
    title: string;
    type: 'success' | 'error' | 'info';
    target: string;
    crdType: string;
  }) => {
    const id = new Date().getTime().toString();
    dispatch(addNotification({ id, message, title, type, target, crdType }));
  };

  return notify;
};
