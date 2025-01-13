import { NOTIFICATION_TYPE } from '@/types';
import { create } from 'zustand';

interface StateValues {
  title: string;
  message: string;
}

interface SseStateValues extends StateValues {
  sseConnecting: boolean;
  sseStatus: NOTIFICATION_TYPE;
}

interface TokenStateValues extends StateValues {
  tokenExpired: boolean;
  tokenExpiring: boolean;
}

interface StoreStateSetters {
  setSseStatus: (state: SseStateValues) => void;
  setTokenStatus: (state: TokenStateValues) => void;
}

export const useConnectionStore = create<SseStateValues & TokenStateValues & StoreStateSetters>((set) => ({
  title: '',
  message: '',

  sseConnecting: true,
  sseStatus: NOTIFICATION_TYPE.DEFAULT,

  tokenExpired: false,
  tokenExpiring: false,

  setSseStatus: (state) => set(state),
  setTokenStatus: (state) => set(state),
}));
