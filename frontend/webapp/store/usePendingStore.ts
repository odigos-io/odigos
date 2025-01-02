import { create } from 'zustand';
import { OVERVIEW_ENTITY_TYPES, WorkloadId } from '@/types';

// This store is used to keep track of pending items that are being created, updated, or deleted.
// This is used for entities that require an SSE event to be sent from the backend after a CRUD action.
// ---
// Imagine a user instruments a few sources, we want to show loading spinners, toasts etc.
// The CLI will finish processing the CRDs and send an SSE event to the frontend.
// The frontend will then remove the pending item from the store and update the UI by refetching the data.
// ---
// This can be used for non-SSE entities (like actions & rules), but it's not necessary as we refetch-instantly in those cases.

export interface PendingItem {
  entityType: OVERVIEW_ENTITY_TYPES;
  entityId?: string | WorkloadId;
}

interface StoreState {
  pendingItems: PendingItem[];
  setPendingItems: (arr: PendingItem[]) => void;
  addPendingItems: (arr: PendingItem[]) => void;
  removePendingItems: (arr: PendingItem[]) => void;
  isThisPending: (item: PendingItem) => boolean;
}

const itemsAreEqual = (item1: PendingItem, item2: PendingItem) => {
  const entityTypesEqual = item1.entityType === item2.entityType;
  const idsEqual =
    typeof item1.entityId === 'string' && typeof item2.entityId === 'string'
      ? item1.entityId === item2.entityId
      : typeof item1.entityId === 'object' && typeof item2.entityId === 'object'
      ? item1.entityId.namespace === item2.entityId.namespace && item1.entityId.name === item2.entityId.name && item1.entityId.kind === item2.entityId.kind
      : !item1.entityId && !item2.entityId;

  return entityTypesEqual && idsEqual;
};

export const usePendingStore = create<StoreState>((set, get) => ({
  pendingItems: [],
  setPendingItems: (arr) => set({ pendingItems: arr }),
  addPendingItems: (arr) => set((state) => ({ pendingItems: state.pendingItems.concat(arr.filter((addItem) => !state.pendingItems.some((existingItem) => itemsAreEqual(existingItem, addItem)))) })),
  removePendingItems: (arr) => set((state) => ({ pendingItems: state.pendingItems.filter((existingItem) => !arr.find((removeItem) => itemsAreEqual(existingItem, removeItem))) })),

  // Pass an item to check if it's in the pending items array.
  // This is used to show loading spinners, toasts etc.
  isThisPending: (item) => {
    const { pendingItems } = get();
    let bool = false;

    for (let i = 0; i < pendingItems.length; i++) {
      const pendingItem = pendingItems[i];
      if (
        pendingItem.entityType === item.entityType &&
        (!item.entityId ||
          (pendingItem.entityType === OVERVIEW_ENTITY_TYPES.SOURCE
            ? !!pendingItem.entityId &&
              !!item.entityId &&
              (pendingItem.entityId as WorkloadId).namespace === (item.entityId as WorkloadId).namespace &&
              (pendingItem.entityId as WorkloadId).name === (item.entityId as WorkloadId).name &&
              (pendingItem.entityId as WorkloadId).kind === (item.entityId as WorkloadId).kind
            : pendingItem.entityId === item.entityId))
      ) {
        bool = true;
        break;
      }
    }

    return bool;
  },
}));
