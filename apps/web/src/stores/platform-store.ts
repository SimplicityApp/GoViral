import { create } from 'zustand'
import { defaultPlatform } from '@/lib/platforms'

interface PlatformState {
  activePlatform: string
  setActivePlatform: (platform: string) => void
}

export const usePlatformStore = create<PlatformState>((set) => ({
  activePlatform: defaultPlatform,
  setActivePlatform: (platform) => {
    document.documentElement.dataset.platform = platform
    set({ activePlatform: platform })
  },
}))
