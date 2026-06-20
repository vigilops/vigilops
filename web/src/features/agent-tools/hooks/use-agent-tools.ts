import { useQuery } from "@tanstack/react-query"

import { apiClient } from "@/lib/api-client"
import type { ToolStat } from "@/features/agent-tools/types"

const STALE_TIME = 1000 * 30

export const agentToolKeys = {
  all: ["agent-tools"] as const,
  stats: ["agent-tools", "stats"] as const,
}

export function useToolStats() {
  return useQuery({
    queryKey: agentToolKeys.stats,
    queryFn: () => apiClient.get<ToolStat[]>("/v1/agent/tools/stats"),
    staleTime: STALE_TIME,
  })
}
