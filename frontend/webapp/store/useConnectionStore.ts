import { create } from 'zustand';

interface StoreState {
  connecting: boolean;
  setConnecting: (bool: boolean) => void;
  active: boolean;
  setActive: (bool: boolean) => void;
  title: string;
  setTitle: (str: string) => void;
  message: string;
  setMessage: (str: string) => void;
}

export const useConnectionStore = create<StoreState>((set) => ({
  connecting: true,
  setConnecting: (bool) => set({ connecting: bool }),
  active: false,
  setActive: (bool) => set({ active: bool }),
  title: '',
  setTitle: (str) => set({ title: str }),
  message: '',
  setMessage: (str) => set({ message: str }),
}));
