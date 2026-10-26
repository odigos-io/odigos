import { create } from 'zustand';
import { BadgeProps } from '@odigos/ui-kit/components';

export enum StatusKeys {
  Token = 'token',
}

export interface StatusValues {
  status?: BadgeProps['status'];
  label?: BadgeProps['label'];
  modalDismissed?: boolean;
}

interface StoreValues {
  [StatusKeys.Token]?: StatusValues;
}

interface StoreSetters {
  setStatusStore: (k: keyof StoreValues, v?: StatusValues) => void;
  resetStatusStore: (k: keyof StoreValues) => void;
  dismissModal: (k: keyof StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  [StatusKeys.Token]: undefined,

  setStatusStore: (k, v) => set({ [k]: v }),
  resetStatusStore: (k) => set({ [k]: undefined }),
  dismissModal: (k) => {
    set((state) => ({
      [k]: {
        ...state[k],
        modalDismissed: true,
      },
    }));
  },
}));
