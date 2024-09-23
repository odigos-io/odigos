import { ConfiguredDestination, K8sActualSource } from '@/types';
import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

export interface IAppStoreState {
  sources: {
    [key: string]: K8sActualSource[];
  };
  namespaceFutureSelectAppsList: { [key: string]: boolean };
  configuredDestinationsList: ConfiguredDestination[];
}

const initialState: IAppStoreState = {
  sources: {},
  namespaceFutureSelectAppsList: {},
  configuredDestinationsList: [],
};

export const appSlice = createSlice({
  name: 'app',
  initialState,
  reducers: {
    setSources: (
      state,
      action: PayloadAction<{ [key: string]: K8sActualSource[] }>
    ) => {
      state.sources = action.payload;
    },
    setNamespaceFutureSelectAppsList: (
      state,
      action: PayloadAction<{ [key: string]: boolean }>
    ) => {
      state.namespaceFutureSelectAppsList = action.payload;
    },
    addConfiguredDestination: (
      state,
      action: PayloadAction<ConfiguredDestination>
    ) => {
      state.configuredDestinationsList.push(action.payload);
    },

    setConfiguredDestinationsList: (
      state,
      action: PayloadAction<ConfiguredDestination[]>
    ) => {
      state.configuredDestinationsList = action.payload;
    },
    resetSources: (state) => {
      state.sources = initialState.sources;
      state.namespaceFutureSelectAppsList =
        initialState.namespaceFutureSelectAppsList;
    },
    resetState: (state) => {
      state.sources = initialState.sources;
      state.namespaceFutureSelectAppsList =
        initialState.namespaceFutureSelectAppsList;
      state.configuredDestinationsList =
        initialState.configuredDestinationsList;
    },
  },
});

// Action creators are generated for each case reducer function
export const {
  setSources,
  setNamespaceFutureSelectAppsList,
  setConfiguredDestinationsList,
  addConfiguredDestination,
  resetState,
  resetSources,
} = appSlice.actions;

export const appReducer = appSlice.reducer;
