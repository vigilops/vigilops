import { Link, createFileRoute } from "@tanstack/react-router"
import { ArrowLeft, TriangleAlert } from "lucide-react"

import { cn } from "@/lib/utils"
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Skeleton } from "@/components/ui/skeleton"
import { LoopBadge } from "@/features/agent-runs/components/loop-badge"
import { RunStatusBadge } from "@/features/agent-runs/components/run-status-badge"
import { StepTimeline } from "@/features/agent-runs/components/step-timeline"
import { formatCost, formatDuration, formatTokens } from "@/lib/format"
import {
  useAgentLoops,
  useAgentRun,
  useAgentSteps,
} from "@/features/agent-runs/hooks/use-agent-runs"
import { useCurrentProject } from "@/features/auth/hooks/use-auth"
import type { AgentRun } from "@/features/agent-runs/types"

export const Route = createFileRoute("/_auth/dashboard/runs/$runId")({
  component: RunDetailPage,
})

function RunDetailPage() {
  const { runId } = Route.useParams()
  const { currentProjectId } = useCurrentProject()
  const run = useAgentRun(currentProjectId, runId)
  const steps = useAgentSteps(currentProjectId, runId)
  const loops = useAgentLoops(currentProjectId, runId)

  return (
    <div className="flex flex-col gap-6">
      <Link
        to="/dashboard/runs"
        className="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="size-4" />
        Agent runs
      </Link>

      {run.isError ? (
        <Alert variant="destructive">
          <TriangleAlert />
          <AlertTitle>Couldn’t load run</AlertTitle>
          <AlertDescription>{run.error.message}</AlertDescription>
        </Alert>
      ) : run.isLoading || !run.data ? (
        <Skeleton className="h-24 w-full rounded-xl" />
      ) : (
        <RunHeader run={run.data} />
      )}

      <div className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold tracking-tight">Decision trace</h2>
        {steps.isLoading || loops.isLoading ? (
          <div className="flex flex-col gap-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-16 w-full rounded-lg" />
            ))}
          </div>
        ) : steps.data?.length ? (
          <StepTimeline steps={steps.data} loops={loops.data ?? []} />
        ) : (
          <p className="text-sm text-muted-foreground">
            No steps recorded for this run.
          </p>
        )}
      </div>
    </div>
  )
}

function MetaItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
        {label}
      </span>
      <span className="font-mono text-sm">{value}</span>
    </div>
  )
}

function RunHeader({ run }: { run: AgentRun }) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="text-xl font-semibold tracking-tight">
          {run.agent_name}
        </h1>
        <RunStatusBadge status={run.status} />
        {run.loop_detected ? <LoopBadge /> : null}
        <span className="font-mono text-xs text-muted-foreground">
          {run.id}
        </span>
      </div>
      <div className="flex flex-wrap items-end gap-8">
        <div className="flex flex-col gap-0.5">
          <span className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            Termination
          </span>
          <TerminationBadge reason={run.termination_reason} />
        </div>
        <MetaItem label="Steps" value={String(run.total_steps)} />
        <MetaItem label="Tokens" value={formatTokens(run.total_tokens)} />
        <MetaItem label="Cost" value={formatCost(run.total_cost_usd)} />
        <MetaItem label="Duration" value={formatDuration(run.duration_ms)} />
      </div>
    </div>
  )
}

// clean = healthy; loop = stuck; everything else = a hard failure.
function TerminationBadge({ reason }: { reason?: string }) {
  if (!reason) return <span className="font-mono text-sm">—</span>
  const tone =
    reason === "clean"
      ? "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
      : reason === "loop_detected"
        ? "bg-amber-500/10 text-amber-600 dark:text-amber-400"
        : "bg-red-500/10 text-red-600 dark:text-red-400"
  return (
    <span
      className={cn(
        "w-fit rounded-full px-2 py-0.5 font-mono text-xs font-semibold",
        tone
      )}
    >
      {reason}
    </span>
  )
}
