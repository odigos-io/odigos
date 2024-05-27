import { addNotification, store } from '@/store';

export const useNotify = () => {
  const dispatch = store.dispatch;

  const notify = (
    message: string,
    title: string,
    target: string,
    event: string,
    type: 'success' | 'error' | 'info'
  ) => {
    const id = new Date().getTime().toString();
    dispatch(addNotification({ id, message, title, type, target }));
  };

  return notify;
};
