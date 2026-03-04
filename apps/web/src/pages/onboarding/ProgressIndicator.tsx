interface Step {
  id: string
  label: string
}

export function ProgressIndicator({
  steps,
  currentIndex,
}: {
  steps: Step[]
  currentIndex: number
}) {
  return (
    <div className="flex items-center justify-center gap-2">
      {steps.map((step, i) => (
        <div key={step.id} className="flex items-center gap-2">
          <div className="flex flex-col items-center">
            <div
              className={`flex h-2.5 w-2.5 rounded-full transition-colors ${
                i === currentIndex
                  ? 'bg-[var(--color-accent)] scale-125'
                  : i < currentIndex
                    ? 'bg-[var(--color-accent)]/50'
                    : 'bg-[var(--color-border)]'
              }`}
            />
            <span
              className={`mt-1.5 text-[10px] transition-colors ${
                i === currentIndex
                  ? 'font-medium text-[var(--color-accent)]'
                  : 'text-[var(--color-text-secondary)]'
              }`}
            >
              {step.label}
            </span>
          </div>
          {i < steps.length - 1 && (
            <div
              className={`mb-4 h-px w-6 transition-colors ${
                i < currentIndex ? 'bg-[var(--color-accent)]/50' : 'bg-[var(--color-border)]'
              }`}
            />
          )}
        </div>
      ))}
    </div>
  )
}
