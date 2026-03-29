import { Moon, SunMedium } from "lucide-react"

import { Button } from "@/components/ui/button"
import { useTheme } from "@/components/theme-provider"

export function ThemeToggle() {
  const { theme, setTheme } = useTheme()
  const nextTheme = theme === "dark" ? "light" : "dark"

  return (
    <Button
      aria-label="Toggle theme"
      onClick={() => setTheme(nextTheme)}
      size="icon-sm"
      type="button"
      variant="outline"
    >
      {theme === "dark" ? (
        <SunMedium className="size-4" />
      ) : (
        <Moon className="size-4" />
      )}
    </Button>
  )
}
