import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect, useState } from "react"
import { Loader2 } from "lucide-react"

import { Brand } from "@/components/brand"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useAuth, useCreateOrg } from "@/features/auth/hooks/use-auth"
import { ApiError } from "@/lib/api-client"

export const Route = createFileRoute("/_auth/onboarding")({
  component: OnboardingPage,
})

function OnboardingPage() {
  const navigate = useNavigate()
  const { user, orgs, isLoading, isError } = useAuth()
  const [name, setName] = useState("")
  const createOrg = useCreateOrg()

  useEffect(() => {
    if (isError) void navigate({ to: "/login" })
  }, [isError, navigate])

  useEffect(() => {
    if (!isLoading && orgs.length > 0) {
      void navigate({
        to: "/dashboard",
        search: { tab: "performance" as const },
      })
    }
  }, [isLoading, orgs, navigate])

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name.trim()) return
    createOrg.mutate(name.trim(), {
      onSuccess: () =>
        void navigate({
          to: "/dashboard",
          search: { tab: "performance" as const },
        }),
    })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-8">
        <div className="flex flex-col items-center gap-3">
          <Brand />
          <div className="text-center">
            <h1 className="text-xl font-semibold">Create your organization</h1>
            {user && (
              <p className="mt-1 text-sm text-muted-foreground">
                Welcome, {user.name || user.email}. Set up your workspace to get
                started.
              </p>
            )}
          </div>
        </div>

        {createOrg.isError && (
          <Alert variant="destructive">
            <AlertDescription>
              {createOrg.error instanceof ApiError
                ? createOrg.error.message
                : "Failed to create organization"}
            </AlertDescription>
          </Alert>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label htmlFor="org-name" className="text-sm font-medium">
              Organization name
            </label>
            <Input
              id="org-name"
              placeholder="Acme Inc."
              value={name}
              onChange={(e) => setName(e.target.value)}
              disabled={createOrg.isPending}
              autoFocus
            />
          </div>
          <Button
            type="submit"
            className="w-full"
            disabled={!name.trim() || createOrg.isPending}
          >
            {createOrg.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating…
              </>
            ) : (
              "Create organization"
            )}
          </Button>
        </form>
      </div>
    </div>
  )
}
