import { create } from 'zustand';

interface StoreStateValues {
  connecting: boolean;
  active: boolean;
  title: string;
  message: string;
}

interface StoreStateSetters {
  setConnectionStore: (state: StoreStateValues) => void;
  setConnecting: (bool: boolean) => void;
  setActive: (bool: boolean) => void;
  setTitle: (str: string) => void;
  setMessage: (str: string) => void;
}

export const useConnectionStore = create<StoreStateValues & StoreStateSetters>((set) => ({
  connecting: true,
  active: false,
  title: '',
  message: '',

  setConnectionStore: (state) => set(state),
  setConnecting: (bool) => set({ connecting: bool }),
  setActive: (bool) => set({ active: bool }),
  setTitle: (str) => set({ title: str }),
  setMessage: (str) => set({ message: str }),
}));
