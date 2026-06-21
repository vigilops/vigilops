import { Outlet, createFileRoute } from "@tanstack/react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { ModeToggle } from "@/components/mode-toogle"
import { TimeWindowSelect } from "@/components/time-window-select"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { TimeWindowProvider } from "@/context/time-window"
import type { TimeWindow } from "@/lib/format"

export const Route = createFileRoute("/dashboard")({
  validateSearch: (s: Record<string, unknown>): { window?: TimeWindow } => {
    const w = s.window
    return w === "24h" || w === "7d" || w === "30d" ? { window: w } : {}
  },
  component: DashboardLayout,
})

function DashboardLayout() {
  return (
    <TimeWindowProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="min-w-0">
          <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
            <SidebarTrigger />
            <Separator orientation="vertical" className="mr-2 h-4" />
            <div className="ml-auto flex items-center gap-2">
              <TimeWindowSelect />
              <ModeToggle />
            </div>
          </header>
          <main className="flex min-w-0 flex-1 flex-col gap-6 overflow-x-hidden p-6">
            <Outlet />
          </main>
        </SidebarInset>
      </SidebarProvider>
    </TimeWindowProvider>
  )
}
