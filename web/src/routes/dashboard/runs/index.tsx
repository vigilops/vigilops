import { useEffect, useState } from "react"
import { Link, createFileRoute } from "@tanstack/react-router"
import { ChevronLeft, ChevronRight } from "lucide-react"

import { EmptyState } from "@/components/empty-state"
import { ErrorState } from "@/components/error-state"
import { TableSkeleton } from "@/components/table-skeleton"
import { Button } from "@/components/ui/button"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { LoopBadge } from "@/features/agent-runs/components/loop-badge"
import { MetricCard } from "@/features/agent-runs/components/metric-card"
import { RunsChart } from "@/features/agent-runs/components/runs-chart"
import { RunStatusBadge } from "@/features/agent-runs/components/run-status-badge"
import {
  formatCost,
  formatDuration,
  formatPercent,
  formatTokens,
  shortId,
} from "@/lib/format"
import { useTimeWindow } from "@/context/time-window"
import {
  useAgentRuns,
  useRunHealth,
} from "@/features/agent-runs/hooks/use-agent-runs"
import type { RunHealthRow } from "@/features/agent-runs/types"

const PAGE_SIZE = 25

export const Route = createFileRoute("/dashboard/runs/")({
  component: RunsPage,
})

function aggregate(rows: RunHealthRow[]) {
  const totalRuns = rows.reduce((a, r) => a + r.total_runs, 0)
  const completed = rows.reduce((a, r) => a + r.completed_runs, 0)
  const loops = rows.reduce((a, r) => a + r.loop_runs, 0)
  const costWeighted = rows.reduce(
    (a, r) => a + (r.avg_cost_usd ?? 0) * r.total_runs,
    0
  )
  return {
    totalRuns,
    completionRate: totalRuns ? completed / totalRuns : 0,
    loopRate: totalRuns ? loops / totalRuns : 0,
    avgCost: totalRuns ? costWeighted / totalRuns : 0,
  }
}

function RunsPage() {
  const { from, bucket } = useTimeWindow()
  const [page, setPage] = useState(0)

  // Reset to first page when the global window changes.
  useEffect(() => setPage(0), [from])

  const runs = useAgentRuns({
    from,
    limit: PAGE_SIZE,
    offset: page * PAGE_SIZE,
  })
  const health = useRunHealth(from)

  const hasNext = (runs.data?.length ?? 0) === PAGE_SIZE

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Agent runs</h1>
        <p className="text-sm text-muted-foreground">
          Every agent invocation, newest first.
        </p>
      </div>

      <MetricCards health={health} />

      <RunsChart from={from} bucket={bucket} />

      {runs.isError ? (
        <ErrorState title="Couldn’t load runs" message={runs.error.message} />
      ) : runs.isLoading ? (
        <TableSkeleton />
      ) : !runs.data?.length ? (
        <EmptyState
          title="No agent runs yet"
          description="Send one with the SDK and it’ll show up here."
        />
      ) : (
        <>
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Run</TableHead>
                <TableHead>Agent</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Steps</TableHead>
                <TableHead className="text-right">Tokens</TableHead>
                <TableHead className="text-right">Cost</TableHead>
                <TableHead className="text-right">Duration</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {runs.data.map((run) => (
                <TableRow key={run.id} className="cursor-pointer">
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    <Link
                      to="/dashboard/runs/$runId"
                      params={{ runId: run.id }}
                    >
                      {shortId(run.id)}
                    </Link>
                  </TableCell>
                  <TableCell>{run.agent_name}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5">
                      <RunStatusBadge status={run.status} />
                      {run.loop_detected ? <LoopBadge /> : null}
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {run.total_steps}
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {formatTokens(run.total_tokens)}
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {formatCost(run.total_cost_usd)}
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {formatDuration(run.duration_ms)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Page {page + 1}</span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              disabled={page === 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
            >
              <ChevronLeft data-icon="inline-start" />
              Previous
            </Button>
            <Button
              variant="outline"
              size="sm"
              disabled={!hasNext}
              onClick={() => setPage((p) => p + 1)}
            >
              Next
              <ChevronRight data-icon="inline-end" />
            </Button>
          </div>
        </div>
        </>
      )}
    </div>
  )
}

function MetricCards({ health }: { health: ReturnType<typeof useRunHealth> }) {
  if (health.isLoading) {
    return (
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    )
  }
  if (health.isError || !health.data) return null

  const agg = aggregate(health.data)
  return (
    <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <MetricCard label="Total runs" value={formatTokens(agg.totalRuns)} />
      <MetricCard
        label="Completion rate"
        value={formatPercent(agg.completionRate)}
      />
      <MetricCard label="Loop rate" value={formatPercent(agg.loopRate)} />
      <MetricCard label="Avg cost / run" value={formatCost(agg.avgCost)} />
    </div>
  )
}
