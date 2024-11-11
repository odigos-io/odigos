import { DropdownOption } from '@/types';
import { create } from 'zustand';

interface StoreState {
  namespace: DropdownOption | undefined;
  setNamespace: (namespace: DropdownOption | undefined) => void;

  types: DropdownOption[];
  setTypes: (types: DropdownOption[]) => void;

  monitors: DropdownOption[];
  setMonitors: (metrics: DropdownOption[]) => void;
}

export const useFilterStore = create<StoreState>((set) => ({
  namespace: undefined,
  setNamespace: (namespace) => set({ namespace }),

  types: [],
  setTypes: (types) => set({ types }),

  monitors: [],
  setMonitors: (monitors) => set({ monitors }),
}));
