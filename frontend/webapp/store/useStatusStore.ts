import { create } from 'zustand';
import { STATUS_TYPE } from '@odigos/ui-kit/types';

interface StoreValues {
  status: STATUS_TYPE;
  title: string;
  message: string;
}

interface StoreSetters {
  setStatusStore: (s: StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  status: STATUS_TYPE.DEFAULT,
  title: 'Connecting...',
  message: '',

  setStatusStore: (s) => set(s),
}));
