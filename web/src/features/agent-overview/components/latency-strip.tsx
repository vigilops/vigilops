import { Card, CardContent } from "@/components/ui/card"

export interface LatencyStat {
  label: string
  value: string
}

export function LatencyStrip({ items }: { items: LatencyStat[] }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-sm font-semibold tracking-tight">Latency</h2>
      <div className="grid grid-cols-3 gap-3">
        {items.map((m) => (
          <Card key={m.label}>
            <CardContent className="flex flex-col gap-1 p-4">
              <span className="text-xs text-muted-foreground">{m.label}</span>
              <span className="font-mono text-xl font-semibold tracking-tight">
                {m.value}
              </span>
            </CardContent>
          </Card>
        ))}
      </div>
    </section>
  )
}
