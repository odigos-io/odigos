import { create } from 'zustand';
import { StatusType } from '@odigos/ui-kit/types';

interface StoreValues {
  status: StatusType;
  title: string;
  message: string;
}

interface StoreSetters {
  setStatusStore: (s: StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  status: StatusType.Default,
  title: 'Connecting...',
  message: '',

  setStatusStore: (s) => set(s),
}));
