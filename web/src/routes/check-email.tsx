import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { Mail } from "lucide-react"

import { Brand } from "@/components/brand"
import { Button } from "@/components/ui/button"
import { redirectIfVerified } from "@/features/auth/guards"
import { useLogout } from "@/features/auth/hooks/use-auth"

export const Route = createFileRoute("/check-email")({
  beforeLoad: redirectIfVerified,
  component: CheckEmailPage,
})

function CheckEmailPage() {
  const navigate = useNavigate()
  const logout = useLogout()

  async function handleLogout() {
    try {
      await logout.mutateAsync()
    } finally {
      void navigate({ to: "/login" })
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <div className="flex justify-center">
          <Brand className="scale-125" />
        </div>
        <div className="flex flex-col items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-full bg-primary/10">
            <Mail className="h-6 w-6 text-primary" />
          </div>
          <h1 className="text-xl font-semibold">Check your email</h1>
          <p className="text-sm text-muted-foreground">
            We sent a verification link to your email address. Click the link to
            activate your account.
          </p>
        </div>
        <p className="text-xs text-muted-foreground">
          Didn't receive it? Check your spam folder.
        </p>
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground"
          onClick={() => void handleLogout()}
        >
          Sign out
        </Button>
      </div>
    </div>
  )
}
