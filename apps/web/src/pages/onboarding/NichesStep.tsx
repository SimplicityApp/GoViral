import { useState, useMemo } from 'react'
import { NicheSelector } from '@/components/shared/NicheSelector'
import { useUpdateConfigMutation } from '@/hooks/useConfig'
import type { AppConfig } from '@/hooks/useConfig'
import { toast } from 'sonner'
import { Check } from 'lucide-react'

export function NichesStep({
  config,
  selectedPlatforms,
}: {
  config: AppConfig | undefined
  selectedPlatforms: Set<string>
}) {
  const [niches, setNiches] = useState<string[]>(config?.niches || [])
  const [linkedinNiches, setLinkedinNiches] = useState<string[]>(config?.linkedin_niches || [])
  const [saved, setSaved] = useState(false)
  const updateConfig = useUpdateConfigMutation()

  const allNiches = useMemo(
    () => [...new Set([...niches, ...linkedinNiches])],
    [niches, linkedinNiches],
  )

  const handleSave = () => {
    updateConfig.mutate(
      { niches, linkedin_niches: linkedinNiches },
      {
        onSuccess: () => {
          toast.success('Niches saved')
          setSaved(true)
        },
        onError: () => toast.error('Failed to save niches'),
      },
    )
  }

  const showX = selectedPlatforms.has('x')
  const showLinkedIn = selectedPlatforms.has('linkedin')

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Pick Your Niches
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Niches help GoViral discover trending content in your areas of interest.
      </p>
      <div className="flex w-full max-w-md flex-col gap-6">
        {showX && (
          <div>
            <h3 className="mb-2 text-sm font-medium text-[var(--color-text)]">
              X Niches
            </h3>
            <NicheSelector
              selected={niches}
              allNiches={allNiches}
              onChange={(v) => { setNiches(v); setSaved(false) }}
              onAddNiche={(tag) => {
                if (!niches.includes(tag)) {
                  setNiches((prev) => [...prev, tag])
                  setSaved(false)
                }
              }}
            />
          </div>
        )}
        {showLinkedIn && (
          <div>
            <h3 className="mb-2 text-sm font-medium text-[var(--color-text)]">
              LinkedIn Niches
            </h3>
            <NicheSelector
              selected={linkedinNiches}
              allNiches={allNiches}
              onChange={(v) => { setLinkedinNiches(v); setSaved(false) }}
              onAddNiche={(tag) => {
                if (!linkedinNiches.includes(tag)) {
                  setLinkedinNiches((prev) => [...prev, tag])
                  setSaved(false)
                }
              }}
            />
          </div>
        )}
        {!showX && !showLinkedIn && (
          <p className="text-sm text-[var(--color-text-secondary)]">
            No platforms selected that use niches. You can skip this step.
          </p>
        )}
        <button
          onClick={handleSave}
          disabled={updateConfig.isPending}
          className="flex items-center justify-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
        >
          {saved ? (
            <>
              <Check size={16} /> Saved
            </>
          ) : updateConfig.isPending ? (
            'Saving...'
          ) : (
            'Save Niches'
          )}
        </button>
      </div>
    </div>
  )
}
