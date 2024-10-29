import { create } from 'zustand';

interface ModalStoreState {
  currentModal: string;
  setCurrentModal: (str: string) => void;
}

export const useModalStore = create<ModalStoreState>((set) => ({
  currentModal: '',
  setCurrentModal: (str) => set({ currentModal: str }),
}));
