import type { LucideIcon } from 'lucide-react'

interface EmptyStateProps {
  icon: LucideIcon
  title: string
  description: string
  action?: { label: string; onClick: () => void }
}

export function EmptyState({ icon: Icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <Icon size={48} className="mb-4 text-[var(--color-text-secondary)]" />
      <h3 className="mb-1 text-lg font-semibold text-[var(--color-text)]">{title}</h3>
      <p className="mb-4 max-w-sm text-sm text-[var(--color-text-secondary)]">
        {description}
      </p>
      {action && (
        <button
          onClick={action.onClick}
          className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
        >
          {action.label}
        </button>
      )}
    </div>
  )
}
