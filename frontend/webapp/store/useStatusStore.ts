import { create } from 'zustand';
import { NOTIFICATION_TYPE } from '@odigos/ui-kit/types';

interface StoreValues {
  status: NOTIFICATION_TYPE;
  title: string;
  message: string;
}

interface StoreSetters {
  setStatusStore: (s: StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  status: NOTIFICATION_TYPE.DEFAULT,
  title: 'Connecting...',
  message: '',

  setStatusStore: (s) => set(s),
}));
