import { Link } from 'react-router-dom'
import { Button } from '@/components/ui/button'

export default function NotFound() {
  return (
    <div className="flex min-h-[60vh] items-center justify-center px-4">
      <div className="text-center">
        <p className="eyebrow">Error 404</p>
        <h2 className="mt-2 text-2xl font-semibold">This page does not exist</h2>
        <p className="mt-2 text-sm text-muted-foreground">
          Check the address, or head back to the console.
        </p>
        <Button asChild size="sm" className="mt-6">
          <Link to="/">Go to sign in</Link>
        </Button>
      </div>
    </div>
  )
}
