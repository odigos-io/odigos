import { create } from 'zustand';
import type { ConfiguredDestination, DestinationInput, K8sActualSource } from '@/types';

export interface IAppState {
  availableSources: { [key: string]: K8sActualSource[] };
  configuredSources: { [key: string]: K8sActualSource[] };
  configuredFutureApps: { [key: string]: boolean };
  configuredDestinations: { stored: ConfiguredDestination; form: DestinationInput }[];
}

interface IAppStateSetters {
  setAvailableSources: (payload: IAppState['availableSources']) => void;
  setConfiguredSources: (payload: IAppState['configuredSources']) => void;
  setConfiguredFutureApps: (payload: IAppState['configuredFutureApps']) => void;

  setConfiguredDestinations: (payload: IAppState['configuredDestinations']) => void;
  addConfiguredDestination: (payload: { stored: ConfiguredDestination; form: DestinationInput }) => void;
  removeConfiguredDestination: (payload: { type: string }) => void;

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
  removeConfiguredDestination: (payload) => set((state) => ({ configuredDestinations: state.configuredDestinations.filter(({ stored }) => stored.type !== payload.type) })),

  resetState: () => set(() => ({ availableSources: {}, configuredSources: {}, configuredFutureApps: {}, configuredDestinations: [] })),
}));

export { useAppStore };
