import { TriangleAlertIcon } from "lucide-react"

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { cn } from "@/lib/utils"

export function ServerUnavailableBanner({
  className,
}: {
  className?: string
}) {
  return (
    <Alert
      className={cn(
        "border-destructive/40 bg-destructive/5 text-foreground",
        className
      )}
    >
      <TriangleAlertIcon className="size-4 text-destructive" />
      <AlertTitle>Local inspector unavailable</AlertTitle>
      <AlertDescription>
        Unable to reach dashboard server, is it running?
      </AlertDescription>
    </Alert>
  )
}
