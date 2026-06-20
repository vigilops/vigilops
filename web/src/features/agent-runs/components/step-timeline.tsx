import { Repeat } from "lucide-react"

import { cn } from "@/lib/utils"
import { decodeToolPayload } from "@/lib/format"
import type { AgentStep, LoopHit, StepType } from "@/features/agent-runs/types"

const KIND: Record<StepType, { label: string; dot: string }> = {
  think: { label: "think", dot: "bg-indigo-500" },
  tool_call: { label: "tool_call", dot: "bg-sky-500" },
  tool_result: { label: "tool_result", dot: "bg-emerald-500" },
  replan: { label: "replan", dot: "bg-amber-500" },
}

function StepNode({ step }: { step: AgentStep }) {
  const kind = KIND[step.step_type]
  const input = decodeToolPayload(step.tool_input)
  const output = decodeToolPayload(step.tool_output)

  return (
    <div className="relative flex gap-3">
      <span
        className={cn(
          "mt-1.5 size-2 shrink-0 rounded-full ring-4 ring-background",
          kind.dot,
        )}
      />
      <div className="flex min-w-0 flex-1 flex-col gap-1 rounded-lg border bg-card p-3">
        <div className="flex items-center gap-2">
          <span className="font-mono text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
            {kind.label}
          </span>
          {step.tool_name ? (
            <span className="truncate font-mono text-sm font-medium">
              {step.tool_name}
            </span>
          ) : null}
          {step.tool_latency_ms != null ? (
            <span className="ml-auto shrink-0 font-mono text-xs text-muted-foreground">
              {step.tool_latency_ms}ms
            </span>
          ) : null}
        </div>
        {step.content ? (
          <p className="whitespace-pre-wrap break-words text-sm text-foreground/90">
            {step.content}
          </p>
        ) : null}
        {input ? (
          <pre className="max-w-full overflow-x-auto rounded bg-muted px-2 py-1 font-mono text-xs text-muted-foreground">
            {input}
          </pre>
        ) : null}
        {output ? (
          <pre className="max-w-full overflow-x-auto rounded bg-muted px-2 py-1 font-mono text-xs text-muted-foreground">
            → {output}
          </pre>
        ) : null}
      </div>
    </div>
  )
}

function LoopBand({ steps, hits }: { steps: AgentStep[]; hits: number }) {
  return (
    <div className="relative rounded-xl border border-dashed border-amber-500/60 bg-amber-500/5 p-3 pt-5">
      <span className="absolute -top-2.5 left-4 flex items-center gap-1.5 rounded-full border border-amber-500/50 bg-background px-2 py-0.5 text-[11px] font-semibold text-amber-600 dark:text-amber-400">
        <Repeat className="size-3" />
        repeated fingerprint × {hits} — agent stuck
      </span>
      <div className="flex flex-col gap-3">
        {steps.map((s) => (
          <StepNode key={s.id} step={s} />
        ))}
      </div>
    </div>
  )
}

type Segment =
  | { loop: false; steps: AgentStep[] }
  | { loop: true; fingerprint: string; hits: number; steps: AgentStep[] }

// Group consecutive steps that share a looping fingerprint into one band.
function segment(steps: AgentStep[], loops: LoopHit[]): Segment[] {
  const hitsByFp = new Map(loops.map((l) => [l.fingerprint, l.hits]))
  const segments: Segment[] = []

  for (const step of steps) {
    const fp = step.input_fingerprint
    const looping = fp != null && hitsByFp.has(fp)
    const prev = segments[segments.length - 1]

    if (looping && fp != null) {
      if (prev && prev.loop && prev.fingerprint === fp) {
        prev.steps.push(step)
      } else {
        segments.push({
          loop: true,
          fingerprint: fp,
          hits: hitsByFp.get(fp)!,
          steps: [step],
        })
      }
    } else {
      if (prev && !prev.loop) {
        prev.steps.push(step)
      } else {
        segments.push({ loop: false, steps: [step] })
      }
    }
  }
  return segments
}

export function StepTimeline({
  steps,
  loops,
}: {
  steps: AgentStep[]
  loops: LoopHit[]
}) {
  const segments = segment(steps, loops)

  return (
    <div className="flex flex-col gap-3 border-l-2 border-muted pl-4">
      {segments.map((seg, i) =>
        seg.loop ? (
          <LoopBand key={i} steps={seg.steps} hits={seg.hits} />
        ) : (
          seg.steps.map((s) => <StepNode key={s.id} step={s} />)
        ),
      )}
    </div>
  )
}
