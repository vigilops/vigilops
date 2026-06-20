import { createFileRoute } from "@tanstack/react-router"

import { EmptyState } from "@/components/empty-state"
import { ErrorState } from "@/components/error-state"
import { TableSkeleton } from "@/components/table-skeleton"
import { RunHealthTable } from "@/features/agent-runs/components/run-health-table"
import { useRunHealth } from "@/features/agent-runs/hooks/use-agent-runs"

export const Route = createFileRoute("/dashboard/agents/")({
  component: AgentsPage,
})

function AgentsPage() {
  const health = useRunHealth()

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Agents</h1>
        <p className="text-sm text-muted-foreground">
          Run health per agent — completion, loops, and cost.
        </p>
      </div>

      {health.isError ? (
        <ErrorState message={health.error.message} />
      ) : health.isLoading ? (
        <TableSkeleton rows={6} />
      ) : !health.data?.length ? (
        <EmptyState title="No runs yet" />
      ) : (
        <RunHealthTable rows={health.data} />
      )}
    </div>
  )
}
