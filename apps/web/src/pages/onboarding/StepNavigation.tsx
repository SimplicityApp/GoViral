import { ChevronLeft, ChevronRight } from 'lucide-react'

export function StepNavigation({
  isFirst,
  isLast,
  onBack,
  onNext,
  onFinish,
}: {
  isFirst: boolean
  isLast: boolean
  onBack: () => void
  onNext: () => void
  onFinish: () => void
}) {
  return (
    <div className="flex items-center justify-between">
      <button
        type="button"
        onClick={onBack}
        disabled={isFirst}
        className="flex items-center gap-1 rounded-[var(--radius-button)] px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)] disabled:invisible"
      >
        <ChevronLeft size={16} /> Back
      </button>
      {isLast ? (
        <button
          type="button"
          onClick={onFinish}
          className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-6 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
        >
          Go to Dashboard
        </button>
      ) : (
        <button
          type="button"
          onClick={onNext}
          className="flex items-center gap-1 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
        >
          {isFirst ? 'Get Started' : 'Next'} <ChevronRight size={16} />
        </button>
      )}
    </div>
  )
}
