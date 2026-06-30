import { Outlet, createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect } from "react"
import { AppSidebar } from "@/components/app-sidebar"
import { ModeToggle } from "@/components/mode-toogle"
import { TimeWindowSelect } from "@/components/time-window-select"
import { OrgSwitcher, ProjectSwitcher } from "@/components/workspace-switcher"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { TimeWindowProvider } from "@/context/time-window"
import { useAuth, useCurrentProject } from "@/features/auth/hooks/use-auth"
import type { TimeWindow } from "@/lib/format"

export const Route = createFileRoute("/_auth/dashboard")({
  validateSearch: (s: Record<string, unknown>): { window?: TimeWindow } => {
    const w = s.window
    return w === "24h" || w === "7d" || w === "30d" ? { window: w } : {}
  },
  component: DashboardLayout,
})

function DashboardLayout() {
  const {
    orgs,
    currentOrg,
    setCurrentOrg,
    isError,
    isLoading: isPending,
  } = useAuth()
  const { projects, currentProject, setCurrentProject } = useCurrentProject()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isPending && !isError && orgs.length === 0) {
      void navigate({ to: "/onboarding" })
    }
  }, [isPending, isError, orgs, navigate])

  if (isPending || isError || orgs.length === 0) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-6 w-6 animate-spin rounded-full border-2 border-primary border-t-transparent" />
      </div>
    )
  }

  return (
    <TimeWindowProvider>
      <SidebarProvider>
        <AppSidebar />
        <SidebarInset className="min-w-0">
          <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
            <SidebarTrigger />
            <Separator orientation="vertical" className="h-4" />
            <OrgSwitcher
              orgs={orgs}
              currentOrg={currentOrg}
              onSelect={setCurrentOrg}
            />
            <ProjectSwitcher
              projects={projects}
              currentProject={currentProject}
              onSelect={setCurrentProject}
            />
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
