import { useQuery } from "@tanstack/react-query"

import { apiClient } from "@/lib/api-client"
import type {
  AgentRun,
  AgentStep,
  LoopHit,
  RunBucket,
  RunHealthRow,
} from "@/features/agent-runs/types"

export { useToolStats } from "@/features/agent-tools/hooks/use-agent-tools"

export type BucketSize = "1h" | "6h" | "1d"

const STALE_TIME = 1000 * 30 // 30s — telemetry is append-heavy, keep it fresh

export interface ListRunsParams {
  from?: string | null
  to?: string
  limit?: number
  offset?: number
}

export const agentRunKeys = {
  all: ["agent-runs"] as const,
  list: (params?: ListRunsParams) => ["agent-runs", "list", params] as const,
  detail: (id: string) => ["agent-runs", "detail", id] as const,
  steps: (id: string) => ["agent-runs", "steps", id] as const,
  loops: (id: string) => ["agent-runs", "loops", id] as const,
  health: (from: string | null) => ["agent-runs", "health", from] as const,
  timeseries: (from: string | null, bucket: BucketSize) =>
    ["agent-runs", "timeseries", from, bucket] as const,
}

export function useAgentRuns(params?: ListRunsParams) {
  return useQuery({
    queryKey: agentRunKeys.list(params),
    queryFn: () => apiClient.get<AgentRun[]>("/v1/agent/runs", { ...params }),
    enabled: params?.from != null,
    staleTime: STALE_TIME,
  })
}

export function useAgentRun(id: string) {
  return useQuery({
    queryKey: agentRunKeys.detail(id),
    queryFn: () => apiClient.get<AgentRun>(`/v1/agent/runs/${id}`),
    enabled: !!id,
    staleTime: STALE_TIME,
  })
}

export function useAgentSteps(id: string) {
  return useQuery({
    queryKey: agentRunKeys.steps(id),
    queryFn: () => apiClient.get<AgentStep[]>(`/v1/agent/runs/${id}/steps`),
    enabled: !!id,
    staleTime: STALE_TIME,
  })
}

export function useAgentLoops(id: string) {
  return useQuery({
    queryKey: agentRunKeys.loops(id),
    queryFn: () => apiClient.get<LoopHit[]>(`/v1/agent/runs/${id}/loops`),
    enabled: !!id,
    staleTime: STALE_TIME,
  })
}

export function useRunsTimeseries(
  from: string | null,
  bucket: BucketSize = "1h",
) {
  return useQuery({
    queryKey: agentRunKeys.timeseries(from, bucket),
    queryFn: () =>
      apiClient.get<RunBucket[]>("/v1/agent/runs/timeseries", {
        from: from!,
        bucket,
      }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}

export function useRunHealth(from: string | null) {
  return useQuery({
    queryKey: agentRunKeys.health(from),
    queryFn: () =>
      apiClient.get<RunHealthRow[]>("/v1/agent/health", { from: from! }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}
