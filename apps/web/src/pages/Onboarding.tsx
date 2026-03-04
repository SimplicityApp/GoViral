import { useNavigate } from 'react-router-dom'
import { useOnboarding } from '@/hooks/useOnboarding'
import { useConfigQuery } from '@/hooks/useConfig'
import { useExtensionCookies } from '@/hooks/useExtensionCookies'
import { defaultPlatform } from '@/lib/platforms'
import { ProgressIndicator } from './onboarding/ProgressIndicator'
import { StepNavigation } from './onboarding/StepNavigation'
import { WelcomeStep } from './onboarding/WelcomeStep'
import { PlatformsStep } from './onboarding/PlatformsStep'
import { AIKeysStep } from './onboarding/AIKeysStep'
import { ConnectXStep } from './onboarding/ConnectXStep'
import { ConnectLinkedInStep } from './onboarding/ConnectLinkedInStep'
import { ConnectGitHubStep } from './onboarding/ConnectGitHubStep'
import { NichesStep } from './onboarding/NichesStep'
import { DoneStep } from './onboarding/DoneStep'
import { InstallExtensionStep } from './onboarding/InstallExtensionStep'

export function Onboarding() {
  const navigate = useNavigate()
  const { data: config } = useConfigQuery()
  const { extension, extracting, extractCookies } = useExtensionCookies()
  const {
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
  } = useOnboarding(config)

  const handleFinish = () => {
    completeOnboarding()
    navigate(`/${defaultPlatform}/dashboard`, { replace: true })
  }

  const handleSkipAll = () => {
    completeOnboarding()
    navigate(`/${defaultPlatform}/dashboard`, { replace: true })
  }

  const renderStep = () => {
    switch (currentStep.id) {
      case 'welcome':
        return <WelcomeStep />
      case 'platforms':
        return (
          <PlatformsStep
            selectedPlatforms={selectedPlatforms}
            onToggle={togglePlatform}
          />
        )
      case 'ai-keys':
        return <AIKeysStep config={config} />
      case 'install-extension':
        return <InstallExtensionStep extension={extension} onRecheck={saveAndReload} />
      case 'connect-x':
        return (
          <ConnectXStep
            config={config}
            extension={extension}
            extensionExtracting={extracting}
            extractCookies={extractCookies}
          />
        )
      case 'connect-linkedin':
        return (
          <ConnectLinkedInStep
            config={config}
            extension={extension}
            extensionExtracting={extracting}
            extractCookies={extractCookies}
          />
        )
      case 'connect-github':
        return <ConnectGitHubStep config={config} />
      case 'niches':
        return <NichesStep config={config} selectedPlatforms={selectedPlatforms} />
      case 'done':
        return <DoneStep />
      default:
        return null
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-[var(--color-bg)]">
      {/* Header */}
      <header className="flex items-center justify-between border-b border-[var(--color-border)] px-6 py-4">
        <span className="text-lg font-bold text-[var(--color-text)]">GoViral</span>
        <button
          onClick={handleSkipAll}
          className="text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
        >
          Skip All
        </button>
      </header>

      {/* Progress */}
      <div className="border-b border-[var(--color-border)] px-6 py-4">
        <ProgressIndicator steps={visibleSteps} currentIndex={currentStepIndex} />
      </div>

      {/* Content */}
      <div className="flex flex-1 items-start justify-center overflow-y-auto px-6 py-12">
        <div className="w-full max-w-lg">{renderStep()}</div>
      </div>

      {/* Navigation */}
      <div className="border-t border-[var(--color-border)] px-6 py-4">
        <div className="mx-auto max-w-lg">
          <StepNavigation
            isFirst={isFirst}
            isLast={isLast}
            onBack={goBack}
            onNext={goNext}
            onFinish={handleFinish}
          />
        </div>
      </div>
    </div>
  )
}
