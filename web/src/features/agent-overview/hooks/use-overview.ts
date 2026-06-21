import { useQuery } from "@tanstack/react-query"

import { apiClient } from "@/lib/api-client"
import type {
  StepTypeCount,
  SummaryResponse,
  TerminationCount,
} from "@/features/agent-overview/types"

const STALE_TIME = 1000 * 30

export const overviewKeys = {
  summary: (from: string | null) => ["overview", "summary", from] as const,
  stepDist: (from: string | null) =>
    ["overview", "step-distribution", from] as const,
  terminations: (from: string | null) =>
    ["overview", "terminations", from] as const,
}

export function useSummary(from: string | null) {
  return useQuery({
    queryKey: overviewKeys.summary(from),
    queryFn: () =>
      apiClient.get<SummaryResponse>("/v1/agent/summary", { from: from! }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}

export function useStepDistribution(from: string | null) {
  return useQuery({
    queryKey: overviewKeys.stepDist(from),
    queryFn: () =>
      apiClient.get<StepTypeCount[]>("/v1/agent/steps/distribution", {
        from: from!,
      }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}

export function useTerminations(from: string | null) {
  return useQuery({
    queryKey: overviewKeys.terminations(from),
    queryFn: () =>
      apiClient.get<TerminationCount[]>("/v1/agent/runs/terminations", {
        from: from!,
      }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}
