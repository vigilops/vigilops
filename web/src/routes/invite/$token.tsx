import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"

import { Brand } from "@/components/brand"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Button } from "@/components/ui/button"
import { inviteApi } from "@/features/auth/api"
import { meQueryOptions, useAuth } from "@/features/auth/hooks/use-auth"
import { ApiError } from "@/lib/api-client"

export const Route = createFileRoute("/invite/$token")({
  component: InvitePage,
})

function InvitePage() {
  const { token } = Route.useParams()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { isAuthenticated } = useAuth()

  const {
    data: invite,
    isPending,
    isError,
  } = useQuery({
    queryKey: ["invite", token],
    queryFn: () => inviteApi.getByToken(token),
    retry: false,
  })

  const accept = useMutation({
    mutationFn: () => inviteApi.accept(token),
    onSuccess: async () => {
      await qc.invalidateQueries({ queryKey: meQueryOptions.queryKey })
      void navigate({
        to: "/dashboard",
        search: { tab: "performance" as const },
      })
    },
  })

  if (isPending) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex min-h-screen items-center justify-center px-4">
        <div className="w-full max-w-sm space-y-6 text-center">
          <Brand />
          <p className="text-sm text-muted-foreground">
            This invite link is invalid, expired, or already used.
          </p>
          <Button
            variant="outline"
            onClick={() => void navigate({ to: "/login" })}
          >
            Go to login
          </Button>
        </div>
      </div>
    )
  }

  const isForbidden =
    accept.isError &&
    accept.error instanceof ApiError &&
    accept.error.status === 403

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-8">
        <div className="flex flex-col items-center gap-3">
          <Brand />
          <p className="text-sm text-muted-foreground">
            You've been invited to join
          </p>
          <p className="text-lg font-semibold">{invite.org_name}</p>
          <p className="text-xs text-muted-foreground">
            as <span className="font-medium">{invite.role}</span>
          </p>
        </div>

        {isForbidden ? (
          <Alert variant="destructive">
            <AlertDescription>
              This invite was not issued to your account. If you believe this is
              an error, contact your organization admin or{" "}
              <a
                href="mailto:support@keelwave.com"
                className="underline underline-offset-2"
              >
                keelwave support
              </a>
              .
            </AlertDescription>
          </Alert>
        ) : accept.isError ? (
          <Alert variant="destructive">
            <AlertDescription>
              {accept.error instanceof ApiError
                ? accept.error.message
                : "Failed to accept invite"}
            </AlertDescription>
          </Alert>
        ) : null}

        {isAuthenticated ? (
          <Button
            className="w-full"
            disabled={accept.isPending || isForbidden}
            onClick={() => accept.mutate()}
          >
            {accept.isPending ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Joining…
              </>
            ) : (
              `Join ${invite.org_name}`
            )}
          </Button>
        ) : (
          <div className="space-y-3">
            <Button
              className="w-full"
              onClick={() =>
                void navigate({
                  to: "/login",
                  search: { redirect: `/invite/${token}` } as never,
                })
              }
            >
              Sign in to accept
            </Button>
            <p className="text-center text-xs text-muted-foreground">
              Don't have an account?{" "}
              <button
                type="button"
                className="underline underline-offset-3 hover:no-underline"
                onClick={() =>
                  void navigate({
                    to: "/login",
                    search: { redirect: `/invite/${token}` } as never,
                  })
                }
              >
                Sign up
              </button>
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
