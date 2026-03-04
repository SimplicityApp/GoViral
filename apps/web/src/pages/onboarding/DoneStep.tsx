import { CheckCircle } from 'lucide-react'

export function DoneStep() {
  return (
    <div className="flex flex-col items-center text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-green-500/15">
        <CheckCircle size={36} className="text-green-400" />
      </div>
      <h2 className="mb-3 text-2xl font-bold text-[var(--color-text)]">
        You're All Set!
      </h2>
      <p className="mb-4 max-w-md text-[var(--color-text-secondary)]">
        GoViral is ready to go. Head to the dashboard to discover trending topics,
        build your persona, and start generating viral content.
      </p>
      <p className="text-xs text-[var(--color-text-secondary)]">
        You can always update your settings later from the Settings page.
      </p>
    </div>
  )
}
