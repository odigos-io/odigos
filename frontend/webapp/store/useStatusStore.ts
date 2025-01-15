import { NOTIFICATION_TYPE } from '@/types';
import { create } from 'zustand';

interface StoreValues {
  status: NOTIFICATION_TYPE;
  title: string;
  message: string;
}

interface StoreSetters {
  setStatusStore: (s: StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  status: NOTIFICATION_TYPE.INFO,
  title: 'Connecting...',
  message: '',

  setStatusStore: (s) => set(s),
}));
