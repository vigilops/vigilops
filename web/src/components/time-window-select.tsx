import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { TIME_WINDOWS } from "@/lib/format"
import type { TimeWindow } from "@/lib/format"
import { useTimeWindow } from "@/context/time-window"

export function TimeWindowSelect() {
  const { window, setWindow } = useTimeWindow()
  return (
    <Select
      value={window}
      onValueChange={(w: TimeWindow | null) => {
        if (w) setWindow(w)
      }}
    >
      <SelectTrigger className="w-[150px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {TIME_WINDOWS.map((w) => (
          <SelectItem key={w.value} value={w.value}>
            {w.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
