export interface ToolStat {
  tool_name: string
  call_count: number
  success_count: number
  fail_count: number
  success_rate: number
  p95_latency_ms: number
}
