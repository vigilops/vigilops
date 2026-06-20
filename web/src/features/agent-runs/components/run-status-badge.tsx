import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import type { RunStatus } from "@/features/agent-runs/types"

const STATUS: Record<RunStatus, { label: string; dot: string }> = {
  completed: { label: "completed", dot: "bg-emerald-500" },
  failed: { label: "failed", dot: "bg-red-500" },
  running: { label: "running", dot: "bg-blue-500 animate-pulse" },
}

export function RunStatusBadge({ status }: { status: RunStatus }) {
  const s = STATUS[status]
  return (
    <Badge variant="outline" className="gap-1.5 font-normal">
      <span className={cn("size-1.5 rounded-full", s.dot)} />
      {s.label}
    </Badge>
  )
}
