import { createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

export interface IAppState {
  sources: any;
}

const initialState: IAppState = {
  sources: {},
};

export const appSlice = createSlice({
  name: 'app',
  initialState,
  reducers: {
    setSources: (state, action: PayloadAction<any>) => {
      state.sources = action.payload;
    },
  },
});

// Action creators are generated for each case reducer function
export const { setSources } = appSlice.actions;

export const appReducer = appSlice.reducer;
