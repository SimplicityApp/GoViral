import { BASE_URL } from '@/lib/api'
import { CheckCircle, Download, RefreshCw } from 'lucide-react'

export function InstallExtensionStep({
  extension,
  onRecheck,
}: {
  extension: { available: boolean; version: string | null }
  onRecheck: () => void
}) {

  if (extension.available) {
    return (
      <div className="flex flex-col items-center">
        <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
          Install Extension
        </h2>
        <div className="mt-4 flex items-center gap-3 rounded-[var(--radius-card)] border border-green-500/30 bg-green-500/10 px-5 py-3">
          <CheckCircle size={20} className="text-green-400" />
          <div>
            <span className="text-sm font-medium text-green-400">
              Extension Installed
            </span>
            {extension.version && (
              <span className="ml-2 text-xs text-[var(--color-text-secondary)]">
                v{extension.version}
              </span>
            )}
          </div>
        </div>
        <p className="mt-4 text-sm text-[var(--color-text-secondary)]">
          You're all set — click Next to continue.
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Install Extension
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        The GoViral browser extension is needed to sync cookies for X and
        LinkedIn. Follow these steps to install it:
      </p>

      <ol className="w-full max-w-md space-y-4 text-sm text-[var(--color-text)]">
        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            1
          </span>
          <div>
            <p className="font-medium">Download the extension</p>
            <a
              href={`${BASE_URL}/extension/download`}
              className="mt-1.5 inline-flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
            >
              <Download size={16} /> Download Extension
            </a>
          </div>
        </li>

        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            2
          </span>
          <p className="font-medium">
            Unzip the downloaded{' '}
            <code className="rounded bg-[var(--color-card)] px-1.5 py-0.5 text-xs">
              goviral-extension.zip
            </code>{' '}
            file
          </p>
        </li>

        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            3
          </span>
          <p className="font-medium">
            Open{' '}
            <code className="rounded bg-[var(--color-card)] px-1.5 py-0.5 text-xs">
              chrome://extensions
            </code>{' '}
            in your browser
          </p>
        </li>

        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            4
          </span>
          <p className="font-medium">
            Enable <strong>Developer mode</strong> (toggle in the top-right corner)
          </p>
        </li>

        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            5
          </span>
          <p className="font-medium">
            Click <strong>Load unpacked</strong> and select the unzipped folder
          </p>
        </li>

        <li className="flex gap-3">
          <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-[var(--color-accent)] text-xs font-bold text-white">
            6
          </span>
          <div>
            <p className="font-medium">Recheck to detect the extension</p>
            <button
              onClick={onRecheck}
              className="mt-1.5 flex items-center gap-2 rounded-[var(--radius-button)] border border-[var(--color-border)] bg-[var(--color-card)] px-4 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-border)]"
            >
              <RefreshCw size={16} />
              Recheck
            </button>
          </div>
        </li>
      </ol>
    </div>
  )
}
