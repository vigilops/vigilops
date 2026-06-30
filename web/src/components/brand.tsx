import { Link } from "@tanstack/react-router"

import { KeelwaveMark } from "@/components/keelwave-mark"
import { cn } from "@/lib/utils"

export function Brand({ className }: { className?: string }) {
  return (
    <Link to="/" className={cn("flex items-center gap-0", className)}>
      <KeelwaveMark className="size-11 text-foreground" />
      <span className="-ml-3 text-2xl font-bold tracking-tight">keelwave</span>
    </Link>
  )
}
