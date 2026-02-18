import { GenerateWorkflow } from '@/components/generate/GenerateWorkflow'

export function Generate() {
  return (
    <div className="mx-auto max-w-3xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Generate Content</h2>
      <GenerateWorkflow />
    </div>
  )
}
