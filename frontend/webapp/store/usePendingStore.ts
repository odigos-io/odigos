import { create } from 'zustand';
import { CRUD } from '@/utils';
import { OVERVIEW_ENTITY_TYPES, WorkloadId } from '@/types';

interface PendingItem {
  id?: string | WorkloadId;
  entityType: OVERVIEW_ENTITY_TYPES;
  crudType: CRUD;
}

interface StoreState {
  pendingItems: PendingItem[];
  setPendingItems: (arr: PendingItem[]) => void;
  addPendingItems: (arr: PendingItem[]) => void;
  removePendingItems: (arr: PendingItem[]) => void;
}

const itemsAreEqual = (item1: PendingItem, item2: PendingItem) => {
  const idsEqual =
    typeof item1.id === 'string' && typeof item2.id === 'string'
      ? item1.id === item2.id
      : typeof item1.id === 'object' && typeof item2.id === 'object'
      ? item1.id.namespace === item2.id.namespace && item1.id.name === item2.id.name && item1.id.kind === item2.id.kind
      : !item1.id && !item2.id;
  const entityTypesEqual = item1.entityType === item2.entityType;
  const crudTypesEqual = item1.crudType === item2.crudType;

  return idsEqual && entityTypesEqual && crudTypesEqual;
};

export const usePendingStore = create<StoreState>((set) => ({
  pendingItems: [],
  setPendingItems: (arr) => set({ pendingItems: arr }),
  addPendingItems: (arr) => set((state) => ({ pendingItems: state.pendingItems.concat(arr.filter((addItem) => !state.pendingItems.some((existingItem) => itemsAreEqual(existingItem, addItem)))) })),
  removePendingItems: (arr) => set((state) => ({ pendingItems: state.pendingItems.filter((existingItem) => !arr.find((removeItem) => itemsAreEqual(existingItem, removeItem))) })),
}));
