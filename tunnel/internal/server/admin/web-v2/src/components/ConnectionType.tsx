import { Badge } from '@/components/ui/badge'
import type { ConnectionType } from '@/types'

interface ConnectionTypeProps {
  type: ConnectionType
}

export default function ConnectionType({ type }: ConnectionTypeProps) {
  return (
    <Badge variant="outline" className="uppercase font-mono text-xs">
      {type}
    </Badge>
  )
}
