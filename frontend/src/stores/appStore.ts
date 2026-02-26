import { create } from 'zustand';
import type { SemesterInfo } from '@/types';
import { semesterApi } from '@/services';

interface AppState {
  currentSemester: SemesterInfo | null;
  sidebarCollapsed: boolean;
  globalLoading: boolean;

  fetchCurrentSemester: () => Promise<void>;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setGlobalLoading: (loading: boolean) => void;
}

export const useAppStore = create<AppState>((set) => ({
  currentSemester: null,
  sidebarCollapsed: false,
  globalLoading: false,

  fetchCurrentSemester: async () => {
    try {
      const { data } = await semesterApi.getCurrent();
      set({ currentSemester: data.data });
    } catch {
      set({ currentSemester: null });
    }
  },

  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
  setGlobalLoading: (loading) => set({ globalLoading: loading }),
}));
