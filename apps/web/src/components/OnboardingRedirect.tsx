import { Navigate } from 'react-router-dom'
import { isOnboardingComplete } from '@/hooks/useOnboarding'
import { defaultPlatform } from '@/lib/platforms'

export function OnboardingRedirect() {
  if (isOnboardingComplete()) {
    return <Navigate to={`/${defaultPlatform}/dashboard`} replace />
  }
  return <Navigate to="/onboarding" replace />
}
