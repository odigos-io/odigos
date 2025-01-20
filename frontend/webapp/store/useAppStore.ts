import { create } from 'zustand';
import type { ConfiguredDestination, DestinationInput, K8sActualSource } from '@/types';

export interface IAppState {
  // in onboarding this is used to keep state of sources that are available for selection in a namespace, in-case user goes back a page (from destinations to sources)
  availableSources: { [key: string]: K8sActualSource[] };
  // in onboarding this is used to keep state of added sources, until end of onboarding
  // in overview this is used to globally select sources for further actions (like uninstrument using multi-source-control component)
  configuredSources: { [key: string]: K8sActualSource[] };
  // in onboarding this is used to keep state of namespaces with future-apps selected, until end of onboarding
  configuredFutureApps: { [key: string]: boolean };
  // in onbaording this is used to keep state of added destinations, until end of onboarding
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

export const useAppStore = create<IAppState & IAppStateSetters>((set) => ({
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
