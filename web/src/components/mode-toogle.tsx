import {  Moon, Sun } from "lucide-react"
import { useTheme } from "@/components/theme-provider"
import { cn } from "@/lib/utils"

const CYCLE = ["light", "dark"] as const
type Theme = (typeof CYCLE)[number]

const ICONS: Record<Theme, React.ReactNode> = {
  light:  <Sun  className="size-3.5" />,
  dark:   <Moon className="size-3.5" />,
}

const LABELS: Record<Theme, string> = {
  light: "Light",
  dark: "Dark",
}

export function ModeToggle({ className }: { className?: string }) {
  const { theme, setTheme } = useTheme()

  const current: Theme = (CYCLE.includes(theme as Theme) ? theme : "system") as Theme

  function cycle() {
    const next = CYCLE[(CYCLE.indexOf(current) + 1) % CYCLE.length]
    setTheme(next)
  }

  return (
    <button
      onClick={cycle}
      title={`Theme: ${LABELS[current]} (click to cycle)`}
      className={cn(
        "flex items-center gap-1.5 px-2.5 py-1.5 border border-border",
        "text-muted-foreground hover:text-foreground hover:border-foreground/30",
        "transition-all text-[10px] font-mono tracking-wide",
        className
      )}
    >
      {ICONS[current]}
      <span className="hidden sm:inline">{LABELS[current]}</span>
    </button>
  )
}
