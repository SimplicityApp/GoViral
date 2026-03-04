import { useState } from 'react'
import { MaskedInput } from '@/components/shared/MaskedInput'
import { useUpdateConfigMutation } from '@/hooks/useConfig'
import type { AppConfig } from '@/hooks/useConfig'
import { toast } from 'sonner'
import { Check } from 'lucide-react'

export function ConnectGitHubStep({ config }: { config: AppConfig | undefined }) {
  const [pat, setPat] = useState(config?.github?.personal_access_token || '')
  const [saved, setSaved] = useState(false)
  const updateConfig = useUpdateConfigMutation()

  const handleSave = () => {
    updateConfig.mutate(
      { github: { personal_access_token: pat } },
      {
        onSuccess: () => {
          toast.success('GitHub token saved')
          setSaved(true)
        },
        onError: () => toast.error('Failed to save GitHub token'),
      },
    )
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Connect GitHub
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Add a Personal Access Token so GoViral can read your repository commits
        for the Code to Post feature. Requires <code className="rounded bg-[var(--color-border)] px-1">repo</code> read scope.
      </p>
      <div className="flex w-full max-w-md flex-col gap-4">
        <MaskedInput
          label="Personal Access Token"
          value={pat}
          onChange={(v) => { setPat(v); setSaved(false) }}
          placeholder="ghp_..."
        />
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
            'Save Token'
          )}
        </button>
      </div>
    </div>
  )
}
