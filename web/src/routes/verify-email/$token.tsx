import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect, useRef, useState } from "react"
import { CheckCircle, Loader2, XCircle } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import { authApi } from "@/features/auth/api"
import { meQueryOptions } from "@/features/auth/hooks/use-auth"
import { useQueryClient } from "@tanstack/react-query"

export const Route = createFileRoute("/verify-email/$token")({
  component: VerifyEmailPage,
})

function VerifyEmailPage() {
  const { token } = Route.useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [status, setStatus] = useState<"pending" | "success" | "error">(
    "pending"
  )
  const [dest, setDest] = useState<"/onboarding" | "/dashboard">("/dashboard")
  const didRun = useRef(false)

  useEffect(() => {
    if (didRun.current) return
    didRun.current = true

    authApi
      .verifyEmail(token)
      .then(async () => {
        await qc.invalidateQueries({ queryKey: meQueryOptions.queryKey })
        const data = await qc.fetchQuery(meQueryOptions).catch(() => null)
        const next =
          data && data.organizations.length === 0 ? "/onboarding" : "/dashboard"
        setDest(next)
        setStatus("success")
      })
      .catch(() => setStatus("error"))
  }, [token, qc])

  useEffect(() => {
    if (status !== "success") return
    const t = setTimeout(() => {
      void navigate({
        to: dest,
        ...(dest === "/dashboard"
          ? { search: { tab: "performance" as const } }
          : {}),
      })
    }, 2000)
    return () => clearTimeout(t)
  }, [status, dest, navigate])

  if (status === "pending") {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (status === "success") {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="w-full max-w-sm space-y-6 text-center">
          <div className="flex justify-center">
            <Brand className="scale-125" />
          </div>
          <div className="flex flex-col items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-full bg-green-500/10">
              <CheckCircle className="h-6 w-6 text-green-500" />
            </div>
            <h1 className="text-xl font-semibold">Email verified!</h1>
            <p className="text-sm text-muted-foreground">
              Your account is active. Redirecting you now…
            </p>
          </div>
          <Button
            className="w-full"
            onClick={() =>
              void navigate({
                to: dest,
                ...(dest === "/dashboard"
                  ? { search: { tab: "performance" as const } }
                  : {}),
              })
            }
          >
            Continue
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <div className="flex justify-center">
          <Brand className="scale-125" />
        </div>
        <div className="flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-destructive/10">
            <XCircle className="h-6 w-6 text-destructive" />
          </div>
          <h1 className="text-xl font-semibold">Link invalid or expired</h1>
          <p className="text-sm text-muted-foreground">
            This verification link has expired or already been used. Sign in to
            request a new one.
          </p>
        </div>
        <Button
          variant="outline"
          className="w-full"
          onClick={() => void navigate({ to: "/login" })}
        >
          Back to login
        </Button>
      </div>
    </div>
  )
}
