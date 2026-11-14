import { useState } from 'react'
import { Button } from '../../../shared/ui/Button.tsx'
import { useResetDemo } from '../api.ts'

/**
 * Restores the demo mission (FR-62). Irreversible for everyone using the demo,
 * so it asks for confirmation first (App. F F.4.1).
 */
export function ResetDemo({ onDone }: { onDone?: () => void }) {
  const reset = useResetDemo()
  const [confirming, setConfirming] = useState(false)

  if (!confirming)
    return (
      <Button className="h-8 px-3 text-sm" onClick={() => setConfirming(true)}>
        Reset demo data
      </Button>
    )
  return (
    <span role="group" aria-label="Confirm demo reset" className="flex flex-wrap items-center gap-2">
      <span className="text-ink">Replace every reading with the original demo mission? This cannot be undone.</span>
      <Button
        variant="command"
        className="h-8 px-3 text-sm"
        disabled={reset.isPending}
        onClick={() =>
          reset.mutate(undefined, {
            onSuccess: () => {
              setConfirming(false)
              onDone?.()
            },
          })
        }
      >
        {reset.isPending ? 'Resetting…' : 'Confirm reset'}
      </Button>
      <Button className="h-8 px-3 text-sm" disabled={reset.isPending} onClick={() => setConfirming(false)}>
        Cancel
      </Button>
      {reset.isError && <span className="text-warning-ink">Reset failed — try again.</span>}
    </span>
  )
}
