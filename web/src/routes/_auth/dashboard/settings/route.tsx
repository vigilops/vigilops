import {
  Link,
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"

import { cn } from "@/lib/utils"

export const Route = createFileRoute("/_auth/dashboard/settings")({
  component: SettingsLayout,
})

const SETTINGS_NAV = [
  { title: "General", to: "/dashboard/settings/org" },
  { title: "Members", to: "/dashboard/settings/members" },
  { title: "Projects & Keys", to: "/dashboard/settings/projects" },
] as const

function SettingsLayout() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your organization, members, and projects.
        </p>
      </div>

      <div className="flex gap-8">
        <nav className="w-44 shrink-0">
          <ul className="flex flex-col gap-0.5">
            {SETTINGS_NAV.map((item) => (
              <li key={item.to}>
                <Link
                  to={item.to}
                  className={cn(
                    "block rounded-md px-3 py-2 text-sm transition-colors hover:bg-accent",
                    pathname.startsWith(item.to)
                      ? "bg-accent font-medium text-foreground"
                      : "text-muted-foreground"
                  )}
                >
                  {item.title}
                </Link>
              </li>
            ))}
          </ul>
        </nav>

        <div className="min-w-0 flex-1">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
