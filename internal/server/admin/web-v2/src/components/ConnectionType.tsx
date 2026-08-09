import type { ConnectionType } from '@/types'

interface ConnectionTypeProps {
  type: ConnectionType
}

export default function ConnectionType({ type }: ConnectionTypeProps) {
  return (
    <span className="data inline-flex items-center rounded-sm border border-border px-1.5 py-0.5 text-[0.65rem] uppercase text-muted-foreground">
      {type}
    </span>
  )
}
