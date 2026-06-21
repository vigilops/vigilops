import { TrendingDown, TrendingUp } from "lucide-react"

import { cn } from "@/lib/utils"

export function CountTrend({
  current,
  previous,
  isNew = false,
}: {
  current: number
  previous: number
  isNew?: boolean
}) {
  if (isNew) {
    return <span className="font-mono text-xs text-muted-foreground">new</span>
  }
  const delta = current - previous
  if (delta === 0) {
    return <span className="font-mono text-xs text-muted-foreground">—</span>
  }
  const up = delta > 0
  const Icon = up ? TrendingUp : TrendingDown
  return (
    <span
      className={cn(
        "inline-flex items-center gap-0.5 font-mono text-xs",
        up
          ? "text-emerald-600 dark:text-emerald-400"
          : "text-red-600 dark:text-red-400"
      )}
    >
      <Icon className="size-3" />
      {up ? "+" : "−"}
      {Math.abs(delta)}
    </span>
  )
}
