import { redirect } from "@tanstack/react-router"
import type { QueryClient } from "@tanstack/react-query"
import { meQueryOptions } from "./hooks/use-auth"

type Ctx = { queryClient: QueryClient }

async function getMe(qc: QueryClient) {
  try {
    return await qc.ensureQueryData(meQueryOptions)
  } catch {
    return null
  }
}

export async function requireVerified({ context }: { context: Ctx }) {
  const me = await getMe(context.queryClient)
  if (!me) throw redirect({ to: "/login" })
  if (!me.user.is_verified) throw redirect({ to: "/check-email" })
  return { me }
}

export async function requireSession({ context }: { context: Ctx }) {
  const me = await getMe(context.queryClient)
  if (!me) throw redirect({ to: "/login" })
  return { me }
}

export async function redirectIfVerified({ context }: { context: Ctx }) {
  const me = await getMe(context.queryClient)
  if (!me) throw redirect({ to: "/login" })
  if (me.user.is_verified) {
    const dest = me.organizations.length === 0 ? "/onboarding" : "/dashboard"
    throw redirect({
      to: dest,
      ...(dest === "/dashboard"
        ? { search: { tab: "performance" as const } }
        : {}),
    })
  }
}

export async function redirectIfAuthenticated({ context }: { context: Ctx }) {
  const me = await getMe(context.queryClient)
  if (!me) return
  if (!me.user.is_verified) throw redirect({ to: "/check-email" })
  const dest = me.organizations.length === 0 ? "/onboarding" : "/dashboard"
  throw redirect({
    to: dest,
    ...(dest === "/dashboard"
      ? { search: { tab: "performance" as const } }
      : {}),
  })
}
