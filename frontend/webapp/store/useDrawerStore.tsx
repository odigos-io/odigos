// drawerStore.ts
import { create } from 'zustand';
import type { ActionDataParsed, ActualDestination, InstrumentationRuleSpec, K8sActualSource, OVERVIEW_ENTITY_TYPES, WorkloadId } from '@/types';

type ItemType = OVERVIEW_ENTITY_TYPES;

export interface DrawerBaseItem {
  id: string | WorkloadId;
  item?: InstrumentationRuleSpec | K8sActualSource | ActionDataParsed | ActualDestination;
  type: ItemType;
  // Add common properties here
}

interface DrawerStoreState {
  selectedItem: DrawerBaseItem | null;
  setSelectedItem: (item: DrawerBaseItem | null) => void;
  isDrawerOpen: boolean;
  openDrawer: () => void;
  closeDrawer: () => void;
}

export const useDrawerStore = create<DrawerStoreState>((set) => ({
  selectedItem: null,
  setSelectedItem: (item) => set({ selectedItem: item }),
  isDrawerOpen: false,
  openDrawer: () => set({ isDrawerOpen: true }),
  closeDrawer: () => set({ isDrawerOpen: false }),
}));
