import { create } from 'zustand';
import { type SVG, StatusType } from '@odigos/ui-kit/types';

interface StoreValues {
  status: StatusType;
  message: string;
  priorityMessage?: boolean;
  leftIcon?: SVG;
}

interface StoreSetters {
  setStatusStore: (s: StoreValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  status: StatusType.Default,
  message: '',
  priorityMessage: false,
  leftIcon: undefined,

  setStatusStore: (s) => set(s),
}));
