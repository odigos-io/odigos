import { create } from 'zustand';

interface Store {
  darkMode: boolean;
  setDarkMode: (bool: boolean) => void;
}

export const useDarkModeStore = create<Store>((set) => ({
  darkMode: true,
  setDarkMode: (bool) => set({ darkMode: bool }),
}));
