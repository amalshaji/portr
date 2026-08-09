import { Monitor, Moon, Sun } from "lucide-react"
import SegmentedControl from "@/components/SegmentedControl"
import { useThemeStore, type ThemePreference } from "@/lib/theme"

const options = [
  { value: "light" as const, label: "Light", content: <Sun className="size-3.5" /> },
  { value: "dark" as const, label: "Dark", content: <Moon className="size-3.5" /> },
  {
    value: "system" as const,
    label: "System",
    content: <Monitor className="size-3.5" />,
  },
]

export default function ThemeToggle({ className }: { className?: string }) {
  const preference = useThemeStore((state) => state.preference)
  const setPreference = useThemeStore((state) => state.setPreference)

  return (
    <SegmentedControl<ThemePreference>
      ariaLabel="Colour theme"
      options={options}
      value={preference}
      onChange={setPreference}
      variant="icon"
      className={className}
    />
  )
}
