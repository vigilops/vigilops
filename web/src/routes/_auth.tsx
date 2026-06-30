import { Outlet, createFileRoute } from "@tanstack/react-router"
import { requireVerified } from "@/features/auth/guards"

export const Route = createFileRoute("/_auth")({
  beforeLoad: requireVerified,
  component: () => <Outlet />,
})
