import { create } from 'zustand';
import { type FetchedSource } from '@/@types';
import { type WorkloadId } from '@odigos/ui-utils';

interface IPaginatedState {
  sources: FetchedSource[];
  sourcesNotFinished: boolean;
}

interface IPaginatedStateSetters {
  setSources: (payload: IPaginatedState['sources']) => void;
  addSources: (payload: IPaginatedState['sources']) => void;
  updateSource: (id: WorkloadId, payload: Partial<IPaginatedState['sources'][0]>) => void;
  removeSource: (id: WorkloadId) => void;
  setSourcesNotFinished: (bool: boolean) => void;
}

export const usePaginatedStore = create<IPaginatedState & IPaginatedStateSetters>((set) => ({
  sources: [],
  setSources: (payload) => set({ sources: payload }),
  addSources: (payload) =>
    set((state) => {
      const prev = [...state.sources];
      const noDuplicates = [
        ...payload.filter((newItem) => !state.sources.find((existingItem) => existingItem.namespace === newItem.namespace && existingItem.name === newItem.name && existingItem.kind === newItem.kind)),
      ];

      prev.push(...noDuplicates);
      return { sources: prev };
    }),
  updateSource: (id, payload) =>
    set((state) => {
      const prev = [...state.sources];
      const foundIdx = prev.findIndex(({ namespace, name, kind }) => namespace === id.namespace && name === id.name && kind === id.kind);

      if (foundIdx !== -1) {
        prev[foundIdx] = { ...prev[foundIdx], ...payload };
      }

      return { sources: prev };
    }),
  removeSource: (id) =>
    set((state) => {
      const prev = [...state.sources];
      const foundIdx = prev.findIndex(({ namespace, name, kind }) => namespace === id.namespace && name === id.name && kind === id.kind);

      if (foundIdx !== -1) {
        prev.splice(foundIdx, 1);
      }

      return { sources: prev };
    }),

  sourcesNotFinished: false,
  setSourcesNotFinished: (bool) => set({ sourcesNotFinished: bool }),
}));
