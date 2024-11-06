import { create } from 'zustand';
import type { Notification } from '@/types';

interface StoreState {
  notifications: Notification[];
  addNotification: (notif: Notification) => void;
  markAsDismissed: (id: string) => void;
  markAsSeen: (id: string) => void;
  removeNotification: (id: string) => void;
}

export const useNotificationStore = create<StoreState>((set) => ({
  notifications: [],

  addNotification: (notif) =>
    set((state) => ({
      notifications: [...state.notifications, notif],
    })),

  markAsDismissed: (id) => {
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
