import { useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'

export function MaskedInput({
  label,
  value,
  onChange,
  required,
  placeholder,
}: {
  label: string
  value: string
  onChange: (v: string) => void
  required?: boolean
  placeholder?: string
}) {
  const [visible, setVisible] = useState(false)
  return (
    <div>
      <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
        {label}
        {required && <span className="text-red-500"> *</span>}
      </label>
      <div className="flex items-center gap-2">
        <input
          type={visible ? 'text' : 'password'}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
        />
        <button
          type="button"
          onClick={() => setVisible(!visible)}
          className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
        >
          {visible ? <EyeOff size={16} /> : <Eye size={16} />}
        </button>
      </div>
    </div>
  )
}
