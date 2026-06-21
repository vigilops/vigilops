export interface ToolStat {
  tool_name: string
  call_count: number
  success_count: number
  fail_count: number
  success_rate: number
  p50_latency_ms: number
  p95_latency_ms: number
  p99_latency_ms: number
  prev_call_count: number
  is_new: boolean
}
