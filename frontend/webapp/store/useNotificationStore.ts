import { create } from 'zustand';
import type { Notification } from '@/types';

interface StoreState {
  notifications: Notification[];
  addNotification: (notif: { type: Notification['type']; title: Notification['title']; message: Notification['message']; crdType: Notification['crdType']; target: Notification['target'] }) => void;
  markAsDismissed: (id?: string) => void;
  markAsSeen: (id?: string) => void;
  removeNotification: (id?: string) => void;
}

export const useNotificationStore = create<StoreState>((set) => ({
  notifications: [],

  addNotification: (notif) =>
    set((state) => {
      const date = new Date();

      return {
        notifications: [
          {
            ...notif,
            id: date.getTime().toString(),
            time: date.toISOString(),
            dismissed: false,
            seen: false,
          },
          ...state.notifications,
        ],
      };
    }),

  markAsDismissed: (id) => {
    if (!id) return;

    set((state) => {
      const foundIdx = state.notifications.findIndex((notif) => notif.id === id);

      if (foundIdx !== -1) {
        state.notifications[foundIdx].dismissed = true;
      }

      return {
        notifications: state.notifications,
      };
    });
  },

  markAsSeen: (id) => {
    if (!id) return;

    set((state) => {
      const foundIdx = state.notifications.findIndex((notif) => notif.id === id);

      if (foundIdx !== -1) {
        state.notifications[foundIdx].seen = true;
      }

      return {
        notifications: state.notifications,
      };
    });
  },

  removeNotification: (id) => {
    if (!id) return;

    set((state) => {
      const foundIdx = state.notifications.findIndex((notif) => notif.id === id);

      if (foundIdx !== -1) {
        state.notifications.splice(foundIdx, 1);
      }

      return {
        notifications: state.notifications,
      };
    });
  },
}));
