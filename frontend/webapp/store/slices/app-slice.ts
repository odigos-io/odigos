import { K8sActualSource } from '@/types';
import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

export interface IAppState {
  sources: {
    [key: string]: K8sActualSource[];
  };
  namespaceFutureSelectAppsList: { [key: string]: boolean };
}

const initialState: IAppState = {
  sources: {},
  namespaceFutureSelectAppsList: {},
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
    // New resetState reducer to reset the state to initial values
    resetState: (state) => {
      state.sources = initialState.sources;
      state.namespaceFutureSelectAppsList =
        initialState.namespaceFutureSelectAppsList;
    },
  },
});

// Action creators are generated for each case reducer function
export const { setSources, setNamespaceFutureSelectAppsList, resetState } =
  appSlice.actions;

export const appReducer = appSlice.reducer;
