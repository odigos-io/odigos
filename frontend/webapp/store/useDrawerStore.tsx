// drawerStore.ts
import { create } from 'zustand';

type ItemType = 'source' | 'action' | 'destination';

interface BaseItem {
  id: string;
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
