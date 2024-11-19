import { create } from 'zustand';
import type { Notification } from '@/types';

interface StoreState {
  notifications: Notification[];
  addNotification: (notif: { type: Notification['type']; title: Notification['title']; message: Notification['message']; crdType: Notification['crdType']; target: Notification['target'] }) => void;
  markAsDismissed: (id: string) => void;
  markAsSeen: (id: string) => void;
  removeNotification: (id: string) => void;
  removeNotifications: (target: string) => void;
}

export const useNotificationStore = create<StoreState>((set) => ({
  notifications: [],

  addNotification: (notif) => {
    const date = new Date();
    const id = `${date.getTime().toString()}${!!notif.target ? `#${notif.target}` : ''}`;

    set((state) => ({
      notifications: [
        {
          ...notif,
          id,
          time: date.toISOString(),
          dismissed: false,
          seen: false,
        },
        ...state.notifications,
      ],
    }));
  },

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

  removeNotifications: (target) => {
    if (!target) return;

    set((state) => {
      const filtered = state.notifications.filter((notif) => notif.id.split('#')[1] !== target);

      return {
        notifications: filtered,
      };
    });
  },
}));
