import { create } from 'zustand';
import type { ConfiguredDestination, K8sActualSource } from '@/types';

export interface IAppState {
  availableSources: { [key: string]: K8sActualSource[] };
  configuredSources: { [key: string]: K8sActualSource[] };
  configuredFutureApps: { [key: string]: boolean };
  configuredDestinations: ConfiguredDestination[];
}

interface IAppStateSetters {
  setAvailableSources: (payload: IAppState['availableSources']) => void;
  setConfiguredSources: (payload: IAppState['configuredSources']) => void;
  setConfiguredFutureApps: (payload: IAppState['configuredFutureApps']) => void;
  setConfiguredDestinations: (payload: IAppState['configuredDestinations']) => void;
  addConfiguredDestination: (payload: ConfiguredDestination) => void;
  resetSources: () => void;
  resetState: () => void;
}

const useAppStore = create<IAppState & IAppStateSetters>((set) => ({
  availableSources: {},
  configuredSources: {},
  configuredFutureApps: {},
  configuredDestinations: [],

  setAvailableSources: (payload) => set({ availableSources: payload }),
  setConfiguredSources: (payload) => set({ configuredSources: payload }),
  setConfiguredFutureApps: (payload) => set({ configuredFutureApps: payload }),
  setConfiguredDestinations: (payload) => set({ configuredDestinations: payload }),
  addConfiguredDestination: (payload) => set((state) => ({ configuredDestinations: [...state.configuredDestinations, payload] })),

  resetSources: () => set(() => ({ availableSources: {}, configuredSources: {}, configuredFutureApps: {} })),
  resetState: () => set(() => ({ availableSources: {}, configuredSources: {}, configuredFutureApps: {}, configuredDestinations: [] })),
}));

export { useAppStore };
