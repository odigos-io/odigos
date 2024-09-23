import { create } from 'zustand';
import { ConfiguredDestination, K8sActualSource } from '@/types';

export interface IAppState {
  sources: { [key: string]: K8sActualSource[] };
  namespaceFutureSelectAppsList: { [key: string]: boolean };
  configuredDestinationsList: ConfiguredDestination[];
}

const useAppStore = create<
  IAppState & {
    setSources: (sources: { [key: string]: K8sActualSource[] }) => void;
    setNamespaceFutureSelectAppsList: (list: {
      [key: string]: boolean;
    }) => void;
    addConfiguredDestination: (destination: ConfiguredDestination) => void;
    setConfiguredDestinationsList: (list: ConfiguredDestination[]) => void;
    resetSources: () => void;
    resetState: () => void;
  }
>((set) => ({
  sources: {},
  namespaceFutureSelectAppsList: {},
  configuredDestinationsList: [],

  setSources: (sources) => set({ sources }),

  setNamespaceFutureSelectAppsList: (list) =>
    set({ namespaceFutureSelectAppsList: list }),

  addConfiguredDestination: (destination) =>
    set((state) => ({
      configuredDestinationsList: [
        ...state.configuredDestinationsList,
        destination,
      ],
    })),

  setConfiguredDestinationsList: (list) =>
    set({ configuredDestinationsList: list }),

  resetSources: () =>
    set((state) => ({
      sources: {},
      namespaceFutureSelectAppsList: {},
    })),

  resetState: () =>
    set(() => ({
      sources: {},
      namespaceFutureSelectAppsList: {},
      configuredDestinationsList: [],
    })),
}));

export { useAppStore };
