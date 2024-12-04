import { create } from 'zustand';

interface StoreState {
  isPolling: boolean;
  togglePolling: (bool?: boolean) => void;
}

export const useBooleanStore = create<StoreState>((set) => ({
  isPolling: false,
  togglePolling: (bool) => set(({ isPolling }) => ({ isPolling: bool ?? !isPolling })),
}));
