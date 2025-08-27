import { create } from 'zustand';
import { WorkloadId } from '@odigos/ui-kit/types';

interface StoreValues {
  holdSourceIds: WorkloadId[];
}

interface StoreSetters {
  setHoldSourceIds: (s: WorkloadId[]) => void;
}

export const useTempHoldStore = create<StoreValues & StoreSetters>((set) => ({
  holdSourceIds: [],
  setHoldSourceIds: (s) => set({ holdSourceIds: s }),
}));
