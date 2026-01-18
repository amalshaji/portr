import { Badge } from '@/components/ui/badge'
import type { ConnectionStatus } from '@/types'

interface ConnectionStatusProps {
  status: ConnectionStatus
}

export default function ConnectionStatus({ status }: ConnectionStatusProps) {
  const getStatusVariant = (status: ConnectionStatus) => {
    switch (status) {
      case 'active':
        return 'default'
      case 'reserved':
        return 'secondary'
      case 'closed':
        return 'outline'
      default:
        return 'outline'
    }
  }

  const getStatusText = (status: ConnectionStatus) => {
    switch (status) {
      case 'active':
        return 'Active'
      case 'reserved':
        return 'Reserved'
      case 'closed':
        return 'Closed'
      default:
        return status
    }
  }

  return (
    <Badge variant={getStatusVariant(status)} className="capitalize">
      {getStatusText(status)}
    </Badge>
  )
}
