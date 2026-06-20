import { createFileRoute } from "@tanstack/react-router"

import { EmptyState } from "@/components/empty-state"
import { ErrorState } from "@/components/error-state"
import { TableSkeleton } from "@/components/table-skeleton"
import { Progress } from "@/components/ui/progress"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatPercent, formatTokens } from "@/lib/format"
import { useToolStats } from "@/features/agent-tools/hooks/use-agent-tools"

export const Route = createFileRoute("/dashboard/tools/")({
  component: ToolsPage,
})

function ToolsPage() {
  const tools = useToolStats()

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-1">
        <h1 className="text-xl font-semibold tracking-tight">Tool analytics</h1>
        <p className="text-sm text-muted-foreground">
          Per-tool reliability across all runs.
        </p>
      </div>

      {tools.isError ? (
        <ErrorState message={tools.error.message} />
      ) : tools.isLoading ? (
        <TableSkeleton />
      ) : !tools.data?.length ? (
        <EmptyState title="No tool calls yet" />
      ) : (
        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Tool</TableHead>
                <TableHead className="text-right">Calls</TableHead>
                <TableHead className="w-[220px]">Success rate</TableHead>
                <TableHead className="text-right">Fails</TableHead>
                <TableHead className="text-right">p95 latency</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tools.data.map((t) => (
                <TableRow key={t.tool_name}>
                  <TableCell className="font-mono text-sm">
                    {t.tool_name}
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {formatTokens(t.call_count)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <Progress value={t.success_rate * 100} className="h-1.5" />
                      <span className="w-12 shrink-0 text-right font-mono text-xs text-muted-foreground">
                        {formatPercent(t.success_rate)}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {t.fail_count}
                  </TableCell>
                  <TableCell className="text-right font-mono text-sm">
                    {t.p95_latency_ms}ms
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
