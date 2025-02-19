import { create } from 'zustand';
import { FetchedAction, FetchedDestination, FetchedInstrumentationRule, type FetchedSource } from '@/@types';
import { ENTITY_TYPES, getEntityId, type WorkloadId } from '@odigos/ui-utils';

interface IPaginatedState {
  sources: FetchedSource[];
  destinations: FetchedDestination[];
  actions: FetchedAction[];
  instrumentationRules: FetchedInstrumentationRule[];
}

type EntityId = string | WorkloadId;
type EntityItems = IPaginatedState['sources'] | IPaginatedState['destinations'] | IPaginatedState['actions'] | IPaginatedState['instrumentationRules'];

interface IPaginatedStateSetters {
  setPaginated: (entityType: ENTITY_TYPES, entities: EntityItems) => void;
  addPaginated: (entityType: ENTITY_TYPES, entities: EntityItems) => void;
  removePaginated: (entityType: ENTITY_TYPES, entityIds: EntityId[]) => void;
}

export const usePaginatedStore = create<IPaginatedState & IPaginatedStateSetters>((set) => ({
  sources: [],
  destinations: [],
  actions: [],
  instrumentationRules: [],

  setPaginated: (entityType, payload) => {
    const KEY =
      entityType === ENTITY_TYPES.SOURCE
        ? 'sources'
        : entityType === ENTITY_TYPES.DESTINATION
        ? 'destinations'
        : entityType === ENTITY_TYPES.ACTION
        ? 'actions'
        : entityType === ENTITY_TYPES.INSTRUMENTATION_RULE
        ? 'instrumentationRules'
        : 'NONE';

    if (KEY === 'NONE') return;

    set({ [KEY]: payload });
  },

  addPaginated: (entityType, entities) => {
    const KEY =
      entityType === ENTITY_TYPES.SOURCE
        ? 'sources'
        : entityType === ENTITY_TYPES.DESTINATION
        ? 'destinations'
        : entityType === ENTITY_TYPES.ACTION
        ? 'actions'
        : entityType === ENTITY_TYPES.INSTRUMENTATION_RULE
        ? 'instrumentationRules'
        : 'NONE';

    if (KEY === 'NONE') return;

    set((state) => {
      const prev = [...state[KEY]];

      entities.forEach((newItem) => {
        const foundIdx = prev.findIndex((oldItem) => JSON.stringify(getEntityId(oldItem)) === JSON.stringify(getEntityId(newItem)));

        if (foundIdx !== -1) {
          prev[foundIdx] = { ...prev[foundIdx], ...newItem };
        } else {
          prev.push(newItem);
        }
      });

      return { [KEY]: prev };
    });
  },

  removePaginated: (entityType, entityIds) => {
    const KEY =
      entityType === ENTITY_TYPES.SOURCE
        ? 'sources'
        : entityType === ENTITY_TYPES.DESTINATION
        ? 'destinations'
        : entityType === ENTITY_TYPES.ACTION
        ? 'actions'
        : entityType === ENTITY_TYPES.INSTRUMENTATION_RULE
        ? 'instrumentationRules'
        : 'NONE';

    if (KEY === 'NONE') return;

    set((state) => {
      const prev = [...state[KEY]];

      entityIds.forEach((id) => {
        const foundIdx = prev.findIndex((entity) => JSON.stringify(getEntityId(entity)) === (entityType === ENTITY_TYPES.SOURCE ? JSON.stringify(getEntityId(id as WorkloadId)) : id));

        if (foundIdx !== -1) {
          prev.splice(foundIdx, 1);
        }
      });

      return { [KEY]: prev };
    });
  },
}));
