const statusStyles: Record<string, string> = {
  draft: 'bg-gray-500/20 text-gray-400',
  approved: 'bg-yellow-500/20 text-yellow-400',
  posted: 'bg-green-500/20 text-green-400',
}

interface StatusBadgeProps {
  status: 'draft' | 'approved' | 'posted'
}

export function StatusBadge({ status }: StatusBadgeProps) {
  return (
    <span
      className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ${statusStyles[status] ?? ''}`}
    >
      {status}
    </span>
  )
}
