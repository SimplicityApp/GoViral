import { useState, useMemo, useEffect, useCallback, useRef } from 'react'
import type { AppConfig } from './useConfig'

const ONBOARDING_KEY = 'goviral_onboarding_complete'
const STEP_RESTORE_KEY = 'goviral_onboarding_step'
const PLATFORMS_RESTORE_KEY = 'goviral_onboarding_platforms'

export function isOnboardingComplete(): boolean {
  return localStorage.getItem(ONBOARDING_KEY) === 'true'
}

export function completeOnboarding(): void {
  localStorage.setItem(ONBOARDING_KEY, 'true')
}

interface StepDef {
  id: string
  label: string
  visibleWhen?: (platforms: Set<string>) => boolean
}

const ALL_STEPS: StepDef[] = [
  { id: 'welcome', label: 'Welcome' },
  { id: 'platforms', label: 'Platforms' },
  { id: 'ai-keys', label: 'AI Keys' },
  { id: 'install-extension', label: 'Extension', visibleWhen: (p) => p.has('x') || p.has('linkedin') },
  { id: 'connect-x', label: 'Connect X', visibleWhen: (p) => p.has('x') },
  { id: 'connect-linkedin', label: 'Connect LinkedIn', visibleWhen: (p) => p.has('linkedin') },
  { id: 'connect-github', label: 'Connect GitHub', visibleWhen: (p) => p.has('github') },
  { id: 'niches', label: 'Niches' },
  { id: 'done', label: 'Done' },
]

function derivePlatformsFromConfig(config?: AppConfig): Set<string> | null {
  if (!config) return null
  const platforms = new Set<string>()
  if (config.x.username || config.x.has_twikit_auth) platforms.add('x')
  if (config.linkedin.has_linkitin_auth) platforms.add('linkedin')
  if (config.github?.personal_access_token) platforms.add('github')
  return platforms.size > 0 ? platforms : null
}

export function useOnboarding(config?: AppConfig) {
  const [currentStepIndex, setCurrentStepIndex] = useState(() => {
    const saved = sessionStorage.getItem(STEP_RESTORE_KEY)
    if (saved) {
      sessionStorage.removeItem(STEP_RESTORE_KEY)
      return Number(saved) || 0
    }
    return 0
  })
  const [selectedPlatforms, setSelectedPlatforms] = useState<Set<string>>(
    () => {
      const saved = sessionStorage.getItem(PLATFORMS_RESTORE_KEY)
      if (saved) {
        sessionStorage.removeItem(PLATFORMS_RESTORE_KEY)
        try {
          const arr = JSON.parse(saved) as string[]
          return new Set(arr)
        } catch { /* ignore */ }
      }
      return new Set(['x', 'linkedin'])
    },
  )
  const hasAppliedConfig = useRef(false)

  // Derive initial platforms from config once when it first loads
  useEffect(() => {
    if (hasAppliedConfig.current || !config) return
    hasAppliedConfig.current = true
    const derived = derivePlatformsFromConfig(config)
    if (derived) {
      setSelectedPlatforms(derived)
    }
  }, [config])

  const visibleSteps = useMemo(
    () => ALL_STEPS.filter((s) => !s.visibleWhen || s.visibleWhen(selectedPlatforms)),
    [selectedPlatforms],
  )

  // Clamp index when platforms change and steps shrink
  useEffect(() => {
    if (currentStepIndex >= visibleSteps.length) {
      setCurrentStepIndex(Math.max(0, visibleSteps.length - 1))
    }
  }, [visibleSteps.length, currentStepIndex])

  const currentStep = visibleSteps[currentStepIndex] ?? visibleSteps[0]

  const goNext = useCallback(() => {
    setCurrentStepIndex((i) => Math.min(i + 1, visibleSteps.length - 1))
  }, [visibleSteps.length])

  const goBack = useCallback(() => {
    setCurrentStepIndex((i) => Math.max(i - 1, 0))
  }, [])

  const isFirst = currentStepIndex === 0
  const isLast = currentStepIndex === visibleSteps.length - 1

  const saveAndReload = useCallback(() => {
    sessionStorage.setItem(STEP_RESTORE_KEY, String(currentStepIndex))
    sessionStorage.setItem(PLATFORMS_RESTORE_KEY, JSON.stringify([...selectedPlatforms]))
    window.location.reload()
  }, [currentStepIndex, selectedPlatforms])

  const togglePlatform = useCallback((platform: string) => {
    setSelectedPlatforms((prev) => {
      const next = new Set(prev)
      if (next.has(platform)) {
        next.delete(platform)
      } else {
        next.add(platform)
      }
      return next
    })
  }, [])

  return {
    currentStep,
    currentStepIndex,
    visibleSteps,
    selectedPlatforms,
    togglePlatform,
    goNext,
    goBack,
    isFirst,
    isLast,
    completeOnboarding,
    saveAndReload,
  }
}
