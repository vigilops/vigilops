import { useQuery } from "@tanstack/react-query"

import { apiClient } from "@/lib/api-client"
import type { ToolStat } from "@/features/agent-tools/types"

const STALE_TIME = 1000 * 30

export const agentToolKeys = {
  all: ["agent-tools"] as const,
  stats: (from: string | null) => ["agent-tools", "stats", from] as const,
}

export function useToolStats(from: string | null) {
  return useQuery({
    queryKey: agentToolKeys.stats(from),
    queryFn: () =>
      apiClient.get<ToolStat[]>("/v1/agent/tools/stats", { from: from! }),
    enabled: from != null,
    staleTime: STALE_TIME,
  })
}
