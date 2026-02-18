import { create } from 'zustand'

interface UIState {
  sidebarOpen: boolean
  workflowStep: number
  setSidebarOpen: (open: boolean) => void
  setWorkflowStep: (step: number) => void
}

export const useUIStore = create<UIState>((set) => ({
  sidebarOpen: false,
  workflowStep: 0,
  setSidebarOpen: (open) => set({ sidebarOpen: open }),
  setWorkflowStep: (step) => set({ workflowStep: step }),
}))
