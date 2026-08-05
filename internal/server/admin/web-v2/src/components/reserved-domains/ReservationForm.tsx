import { useMemo, useState } from "react"
import { LoaderCircle, Plus } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Progress } from "@/components/ui/progress"
import { normalizeSubdomain, validateSubdomain } from "@/lib/subdomain"

interface ReservationFormProps {
  baseDomain: string
  count: number
  limit: number
  loading: boolean
  submitting: boolean
  onReserve: (subdomain: string) => Promise<void>
}

export function ReservationForm({
  baseDomain,
  count,
  limit,
  loading,
  submitting,
  onReserve,
}: ReservationFormProps) {
  const [subdomain, setSubdomain] = useState("")
  const [inputTouched, setInputTouched] = useState(false)
  const [submitError, setSubmitError] = useState("")
  const normalizedSubdomain = normalizeSubdomain(subdomain)
  const validationError = validateSubdomain(subdomain)
  const atLimit = limit === 0 || count >= limit
  const quotaPercent = useMemo(() => {
    if (limit === 0) return 100
    return Math.min(100, (count / limit) * 100)
  }, [count, limit])

  const reserve = async (event: React.FormEvent) => {
    event.preventDefault()
    setInputTouched(true)
    setSubmitError("")
    if (validationError || atLimit) return

    try {
      await onReserve(normalizedSubdomain)
      setSubdomain("")
      setInputTouched(false)
    } catch (error) {
      setSubmitError(
        error instanceof Error ? error.message : "Subdomain could not be reserved",
      )
    }
  }

  return (
    <div className="space-y-5 p-5 sm:p-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h2 className="font-medium">Reserve a subdomain</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Only your current team credential can use a name reserved here.
          </p>
        </div>
        <p className="shrink-0 font-mono text-xs text-muted-foreground">
          {count} / {limit}
        </p>
      </div>

      <Progress
        value={quotaPercent}
        aria-label={`${count} of ${limit} reserved subdomains used`}
        className="h-1.5 bg-muted [&_[data-slot=progress-indicator]]:transition-transform motion-reduce:[&_[data-slot=progress-indicator]]:transition-none"
      />

      <form onSubmit={reserve} className="space-y-2">
        <label htmlFor="reserved-subdomain" className="text-sm font-medium">
          Subdomain
        </label>
        <div className="flex flex-col gap-2 sm:flex-row">
          <div
            className={`flex min-w-0 flex-1 overflow-hidden rounded-md border bg-background transition-shadow focus-within:border-ring focus-within:ring-3 focus-within:ring-ring/50 ${
              inputTouched && validationError
                ? "border-destructive focus-within:border-destructive focus-within:ring-destructive/20"
                : ""
            }`}
          >
            <input
              id="reserved-subdomain"
              value={subdomain}
              onChange={(event) => {
                setSubdomain(event.target.value.toLowerCase())
                setSubmitError("")
              }}
              onBlur={() => setInputTouched(true)}
              disabled={loading || submitting || atLimit}
              aria-invalid={inputTouched && Boolean(validationError)}
              aria-describedby="reserved-subdomain-help"
              autoComplete="off"
              spellCheck={false}
              placeholder="my-project"
              className="h-10 min-w-0 flex-1 bg-transparent px-3 font-mono text-sm outline-none placeholder:font-sans placeholder:text-muted-foreground disabled:cursor-not-allowed disabled:opacity-50"
            />
            <span className="flex max-w-[55%] items-center border-l bg-muted/50 px-3 font-mono text-sm text-muted-foreground">
              .{baseDomain || "domain"}
            </span>
          </div>
          <Button
            type="submit"
            className="h-10 sm:min-w-28"
            disabled={loading || submitting || atLimit || Boolean(validationError)}
          >
            {submitting ? <LoaderCircle className="animate-spin" /> : <Plus />}
            Reserve
          </Button>
        </div>
        <div id="reserved-subdomain-help" className="min-h-5 text-xs">
          {inputTouched && validationError ? (
            <p className="text-destructive">{validationError}</p>
          ) : submitError ? (
            <p className="text-destructive">{submitError}</p>
          ) : atLimit ? (
            <p className="text-muted-foreground">
              {limit === 0
                ? "New reservations are disabled on this server."
                : "Release a name before reserving another."}
            </p>
          ) : (
            <p className="text-muted-foreground">
              Lowercase letters, numbers, and internal hyphens only.
            </p>
          )}
        </div>
      </form>
    </div>
  )
}
