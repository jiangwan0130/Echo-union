import { create } from 'zustand';
import type { SemesterInfo, SemesterPhase, PendingTodoItem } from '@/types';
import { semesterApi, notificationApi } from '@/services';

interface AppState {
  currentSemester: SemesterInfo | null;
  currentPhase: SemesterPhase | null;
  pendingTodos: PendingTodoItem[];
  sidebarCollapsed: boolean;
  globalLoading: boolean;

  fetchCurrentSemester: () => Promise<void>;
  fetchPendingTodos: () => Promise<void>;
  setSidebarCollapsed: (collapsed: boolean) => void;
  setGlobalLoading: (loading: boolean) => void;
}

export const useAppStore = create<AppState>((set) => ({
  currentSemester: null,
  currentPhase: null,
  pendingTodos: [],
  sidebarCollapsed: false,
  globalLoading: false,

  fetchCurrentSemester: async () => {
    try {
      const { data } = await semesterApi.getCurrent();
      set({
        currentSemester: data.data,
        currentPhase: data.data?.phase ?? null,
      });
    } catch {
      set({ currentSemester: null, currentPhase: null });
    }
  },

  fetchPendingTodos: async () => {
    try {
      const { data } = await notificationApi.getPending();
      set({ pendingTodos: data.data?.list ?? [] });
    } catch {
      set({ pendingTodos: [] });
    }
  },

  setSidebarCollapsed: (collapsed) => set({ sidebarCollapsed: collapsed }),
  setGlobalLoading: (loading) => set({ globalLoading: loading }),
}));
