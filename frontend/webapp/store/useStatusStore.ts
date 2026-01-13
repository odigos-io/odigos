import { create } from 'zustand';
import { BadgeProps } from '@odigos/ui-kit/components/v2';

export enum StatusKeys {
  Token = 'token',
  Backend = 'backend',
  Instrumentation = 'instrumentation',
}

export interface StatusValues extends BadgeProps {
  tooltip?: string;
}

interface StoreValues {
  [StatusKeys.Token]?: StatusValues;
  [StatusKeys.Backend]?: StatusValues;
  [StatusKeys.Instrumentation]?: StatusValues;
}

interface StoreSetters {
  setStatusStore: (k: keyof StoreValues, v?: StatusValues) => void;
}

export const useStatusStore = create<StoreValues & StoreSetters>((set) => ({
  [StatusKeys.Token]: undefined,
  [StatusKeys.Backend]: undefined,
  [StatusKeys.Instrumentation]: undefined,

  setStatusStore: (k, v) => set({ [k]: v }),
}));
