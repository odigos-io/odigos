import { create } from 'zustand';
import type { DropdownOption } from '@/types';

export interface FiltersState {
  namespace: DropdownOption | undefined;
  types: DropdownOption[];
  monitors: DropdownOption[];
}

interface StoreState {
  namespace: FiltersState['namespace'];
  setNamespace: (namespace: FiltersState['namespace']) => void;

  types: FiltersState['types'];
  setTypes: (types: FiltersState['types']) => void;

  monitors: FiltersState['monitors'];
  setMonitors: (metrics: FiltersState['monitors']) => void;

  setAll: (params: FiltersState) => void;
  clearAll: () => void;
}

export const useFilterStore = create<StoreState>((set) => ({
  namespace: undefined,
  setNamespace: (namespace) => set({ namespace }),

  types: [],
  setTypes: (types) => set({ types }),

  monitors: [],
  setMonitors: (monitors) => set({ monitors }),

  setAll: (params) => set(params),
  clearAll: () => set({ namespace: undefined, types: [], monitors: [] }),
}));
