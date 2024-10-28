// drawerStore.ts
import { create } from 'zustand';
import { ActionDataParsed, ActualDestination, K8sActualSource, WorkloadId } from '@/types';

type ItemType = 'source' | 'action' | 'destination';

export interface BaseItem {
  id: string | WorkloadId;
  item?: K8sActualSource | ActionDataParsed | ActualDestination;
  type: ItemType;
  // Add common properties here
}

interface DrawerStoreState {
  selectedItem: BaseItem | null;
  setSelectedItem: (item: BaseItem | null) => void;
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
