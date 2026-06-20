import { Repeat } from "lucide-react"

import { Badge } from "@/components/ui/badge"

export function LoopBadge() {
  return (
    <Badge
      variant="outline"
      className="gap-1 border-amber-500/40 font-normal text-amber-600 dark:text-amber-400"
    >
      <Repeat className="size-3" />
      loop
    </Badge>
  )
}
