import { create } from 'zustand';

interface ModalStoreState {
  // isOpen: boolean;
  // toggleOpen: (bool?: boolean) => void;
  currentModal: string;
  setCurrentModal: (str: string) => void;
}

export const useModalStore = create<ModalStoreState>((set) => ({
  // isOpen: false,
  // toggleOpen: (bool) => set((prev) => ({ isOpen: typeof bool === 'boolean' ? bool : !prev.isOpen })),
  currentModal: '',
  setCurrentModal: (str) => set({ currentModal: str }),
}));
