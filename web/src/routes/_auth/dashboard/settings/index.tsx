import { createFileRoute, redirect } from "@tanstack/react-router"

export const Route = createFileRoute("/_auth/dashboard/settings/")({
  beforeLoad: () => {
    throw redirect({ to: "/dashboard/settings/org" })
  },
  component: () => null,
})
