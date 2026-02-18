import { useState, useMemo } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface ScheduleCalendarProps {
  scheduledAt: string
  onChange: (v: string) => void
}

const DAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']
const HOURS = Array.from({ length: 24 }, (_, i) => i)
const MINUTES = [0, 15, 30, 45]

function pad(n: number) {
  return String(n).padStart(2, '0')
}

function toLocalDatetime(date: Date) {
  const y = date.getFullYear()
  const m = pad(date.getMonth() + 1)
  const d = pad(date.getDate())
  const h = pad(date.getHours())
  const min = pad(date.getMinutes())
  return `${y}-${m}-${d}T${h}:${min}`
}

function isSameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

function isBeforeToday(date: Date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const d = new Date(date)
  d.setHours(0, 0, 0, 0)
  return d < today
}

export function ScheduleCalendar({ scheduledAt, onChange }: ScheduleCalendarProps) {
  const now = new Date()
  const selected = scheduledAt ? new Date(scheduledAt) : null

  const [viewYear, setViewYear] = useState(selected?.getFullYear() ?? now.getFullYear())
  const [viewMonth, setViewMonth] = useState(selected?.getMonth() ?? now.getMonth())

  const selectedHour = selected?.getHours() ?? 9
  const selectedMinute = selected?.getMinutes() ?? 0

  const calendarDays = useMemo(() => {
    const firstDay = new Date(viewYear, viewMonth, 1).getDay()
    const daysInMonth = new Date(viewYear, viewMonth + 1, 0).getDate()
    const daysInPrevMonth = new Date(viewYear, viewMonth, 0).getDate()

    const cells: { date: Date; current: boolean }[] = []

    // Previous month trailing days
    for (let i = firstDay - 1; i >= 0; i--) {
      cells.push({ date: new Date(viewYear, viewMonth - 1, daysInPrevMonth - i), current: false })
    }
    // Current month
    for (let d = 1; d <= daysInMonth; d++) {
      cells.push({ date: new Date(viewYear, viewMonth, d), current: true })
    }
    // Next month leading days
    const remaining = 42 - cells.length
    for (let d = 1; d <= remaining; d++) {
      cells.push({ date: new Date(viewYear, viewMonth + 1, d), current: false })
    }

    return cells
  }, [viewYear, viewMonth])

  const monthLabel = new Date(viewYear, viewMonth).toLocaleString('default', { month: 'long', year: 'numeric' })

  function prevMonth() {
    if (viewMonth === 0) {
      setViewMonth(11)
      setViewYear(viewYear - 1)
    } else {
      setViewMonth(viewMonth - 1)
    }
  }

  function nextMonth() {
    if (viewMonth === 11) {
      setViewMonth(0)
      setViewYear(viewYear + 1)
    } else {
      setViewMonth(viewMonth + 1)
    }
  }

  function selectDate(date: Date) {
    const d = new Date(date)
    d.setHours(selectedHour, selectedMinute, 0, 0)
    onChange(toLocalDatetime(d))
  }

  function setTime(hour: number, minute: number) {
    const base = selected ? new Date(selected) : new Date()
    if (!selected) {
      // Default to today if no date selected yet
      base.setHours(hour, minute, 0, 0)
    } else {
      base.setHours(hour, minute, 0, 0)
    }
    onChange(toLocalDatetime(base))
  }

  function applyPreset(offsetMs: number) {
    const d = new Date(Date.now() + offsetMs)
    // Round minutes to nearest 15
    d.setMinutes(Math.ceil(d.getMinutes() / 15) * 15, 0, 0)
    onChange(toLocalDatetime(d))
    setViewYear(d.getFullYear())
    setViewMonth(d.getMonth())
  }

  function applyTomorrowPreset(hour: number) {
    const d = new Date()
    d.setDate(d.getDate() + 1)
    d.setHours(hour, 0, 0, 0)
    onChange(toLocalDatetime(d))
    setViewYear(d.getFullYear())
    setViewMonth(d.getMonth())
  }

  return (
    <div className="flex flex-col gap-4">
      <label className="block text-sm font-medium text-[var(--color-text)]">
        Schedule for
      </label>

      {/* Quick presets */}
      <div className="flex flex-wrap gap-2">
        {[
          { label: 'In 1 hour', action: () => applyPreset(60 * 60 * 1000) },
          { label: 'In 3 hours', action: () => applyPreset(3 * 60 * 60 * 1000) },
          { label: 'Tomorrow 9am', action: () => applyTomorrowPreset(9) },
          { label: 'Tomorrow 12pm', action: () => applyTomorrowPreset(12) },
        ].map((preset) => (
          <button
            key={preset.label}
            type="button"
            onClick={preset.action}
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs font-medium text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)]"
          >
            {preset.label}
          </button>
        ))}
      </div>

      <div className="flex gap-4">
        {/* Calendar */}
        <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-3">
          {/* Month navigation */}
          <div className="mb-2 flex items-center justify-between">
            <button
              type="button"
              onClick={prevMonth}
              className="rounded-[var(--radius-button)] p-1 text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
            >
              <ChevronLeft size={16} />
            </button>
            <span className="text-sm font-medium text-[var(--color-text)]">{monthLabel}</span>
            <button
              type="button"
              onClick={nextMonth}
              className="rounded-[var(--radius-button)] p-1 text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
            >
              <ChevronRight size={16} />
            </button>
          </div>

          {/* Day headers */}
          <div className="grid grid-cols-7 gap-0.5 text-center">
            {DAYS.map((d) => (
              <div key={d} className="py-1 text-xs font-medium text-[var(--color-text-secondary)]">
                {d}
              </div>
            ))}
          </div>

          {/* Day cells */}
          <div className="grid grid-cols-7 gap-0.5">
            {calendarDays.map(({ date, current }, i) => {
              const past = isBeforeToday(date)
              const isToday = isSameDay(date, now)
              const isSelected = selected && isSameDay(date, selected)

              return (
                <button
                  key={i}
                  type="button"
                  disabled={past}
                  onClick={() => selectDate(date)}
                  className={`flex h-8 w-8 items-center justify-center rounded-full text-xs transition-colors ${
                    isSelected
                      ? 'bg-[var(--color-accent)] font-semibold text-white'
                      : isToday
                        ? 'border border-[var(--color-accent)] font-medium text-[var(--color-accent)]'
                        : past
                          ? 'cursor-not-allowed text-[var(--color-text-secondary)]/30'
                          : current
                            ? 'text-[var(--color-text)] hover:bg-[var(--color-card-hover)]'
                            : 'text-[var(--color-text-secondary)]/50 hover:bg-[var(--color-card-hover)]'
                  }`}
                >
                  {date.getDate()}
                </button>
              )
            })}
          </div>
        </div>

        {/* Time picker */}
        <div className="flex flex-col gap-3 rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-3">
          <span className="text-xs font-medium text-[var(--color-text-secondary)]">Time</span>

          {/* Hour */}
          <div className="flex flex-col gap-1">
            <span className="text-xs text-[var(--color-text-secondary)]">Hour</span>
            <div className="max-h-40 overflow-y-auto rounded-[var(--radius-input)] border border-[var(--color-border)]">
              {HOURS.map((h) => (
                <button
                  key={h}
                  type="button"
                  onClick={() => setTime(h, selectedMinute)}
                  className={`block w-full px-3 py-1 text-left text-xs transition-colors ${
                    h === selectedHour
                      ? 'bg-[var(--color-accent)] font-medium text-white'
                      : 'text-[var(--color-text)] hover:bg-[var(--color-card-hover)]'
                  }`}
                >
                  {pad(h)}:00
                </button>
              ))}
            </div>
          </div>

          {/* Minute */}
          <div className="flex flex-col gap-1">
            <span className="text-xs text-[var(--color-text-secondary)]">Minute</span>
            <div className="flex gap-1">
              {MINUTES.map((m) => (
                <button
                  key={m}
                  type="button"
                  onClick={() => setTime(selectedHour, m)}
                  className={`rounded-[var(--radius-button)] px-2.5 py-1 text-xs font-medium transition-colors ${
                    m === selectedMinute
                      ? 'bg-[var(--color-accent)] text-white'
                      : 'border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
                  }`}
                >
                  :{pad(m)}
                </button>
              ))}
            </div>
          </div>

          {/* Selected summary */}
          {selected && (
            <div className="mt-auto border-t border-[var(--color-border)] pt-2 text-xs text-[var(--color-text-secondary)]">
              {selected.toLocaleString('default', {
                weekday: 'short',
                month: 'short',
                day: 'numeric',
                hour: 'numeric',
                minute: '2-digit',
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
