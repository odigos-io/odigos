import { create } from 'zustand';
import { ENTITY_TYPES, getEntityId, type WorkloadId } from '@odigos/ui-utils';
import type { FetchedAction, FetchedDestination, FetchedInstrumentationRule, FetchedSource } from '@/@types';

interface IPaginatedState {
  sources: FetchedSource[];
  sourcesPaginating: boolean;
  sourcesExpected: number;
  destinations: FetchedDestination[];
  destinationsPaginating: boolean;
  destinationsExpected: number;
  actions: FetchedAction[];
  actionsPaginating: boolean;
  actionsExpected: number;
  instrumentationRules: FetchedInstrumentationRule[];
  instrumentationRulesPaginating: boolean;
  instrumentationRulesExpected: number;
}

type EntityId = string | WorkloadId;
type EntityItems = IPaginatedState['sources'] | IPaginatedState['destinations'] | IPaginatedState['actions'] | IPaginatedState['instrumentationRules'];

interface IPaginatedStateSetters {
  setPaginating: (entityType: ENTITY_TYPES, bool: boolean) => void;
  setExpected: (entityType: ENTITY_TYPES, num: number) => void;
  setPaginated: (entityType: ENTITY_TYPES, entities: EntityItems) => void;
  addPaginated: (entityType: ENTITY_TYPES, entities: EntityItems) => void;
  removePaginated: (entityType: ENTITY_TYPES, entityIds: EntityId[]) => void;
}

export const usePaginatedStore = create<IPaginatedState & IPaginatedStateSetters>((set) => ({
  sources: [],
  sourcesPaginating: false,
  sourcesExpected: 0,

  destinations: [],
  destinationsPaginating: false,
  destinationsExpected: 0,

  actions: [],
  actionsPaginating: false,
  actionsExpected: 0,

  instrumentationRules: [],
  instrumentationRulesPaginating: false,
  instrumentationRulesExpected: 0,

  setExpected: (entityType, num) => {
    const KEY =
      entityType === ENTITY_TYPES.SOURCE
        ? 'sourcesExpected'
        : entityType === ENTITY_TYPES.DESTINATION
        ? 'destinationsExpected'
        : entityType === ENTITY_TYPES.ACTION
        ? 'actionsExpected'
        : entityType === ENTITY_TYPES.INSTRUMENTATION_RULE
        ? 'instrumentationRulesExpected'
        : 'NONE';

    if (KEY === 'NONE') return;

    set({ [KEY]: num });
  },

  setPaginating: (entityType, bool) => {
    const KEY =
      entityType === ENTITY_TYPES.SOURCE
        ? 'sourcesPaginating'
        : entityType === ENTITY_TYPES.DESTINATION
        ? 'destinationsPaginating'
        : entityType === ENTITY_TYPES.ACTION
        ? 'actionsPaginating'
        : entityType === ENTITY_TYPES.INSTRUMENTATION_RULE
        ? 'instrumentationRulesPaginating'
        : 'NONE';

    if (KEY === 'NONE') return;

    set({ [KEY]: bool });
  },

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
