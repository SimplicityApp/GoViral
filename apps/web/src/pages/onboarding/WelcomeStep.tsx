import { Sparkles } from 'lucide-react'

export function WelcomeStep() {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-[var(--color-accent)]/15">
        <Sparkles size={32} className="text-[var(--color-accent)]" />
      </div>
      <h1 className="mb-3 text-2xl font-bold text-[var(--color-text)]">
        Welcome to GoViral
      </h1>
      <p className="mb-6 max-w-md text-[var(--color-text-secondary)]">
        Create viral content for X, LinkedIn, and more. We'll help you set up
        your AI keys, connect your accounts, and pick your niches in just a few
        steps.
      </p>
      <div className="grid max-w-sm gap-3 text-left text-sm text-[var(--color-text-secondary)]">
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)]/15 text-xs font-semibold text-[var(--color-accent)]">
            1
          </span>
          <span>Connect your AI providers (Claude, Gemini)</span>
        </div>
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)]/15 text-xs font-semibold text-[var(--color-accent)]">
            2
          </span>
          <span>Link your social accounts for posting</span>
        </div>
        <div className="flex items-start gap-3">
          <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)]/15 text-xs font-semibold text-[var(--color-accent)]">
            3
          </span>
          <span>Pick your niches to discover trending content</span>
        </div>
      </div>
    </div>
  )
}
