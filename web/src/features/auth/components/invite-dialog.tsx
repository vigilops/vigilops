import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Copy, Loader2, Trash2, UserPlus, X } from "lucide-react"
import { useEffect, useRef, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { inviteApi } from "@/features/auth/api"
import { ApiError } from "@/lib/api-client"
import { cn } from "@/lib/utils"

interface InviteDialogProps {
  orgId: string
  orgName: string
  onClose: () => void
}

export function InviteDialog({ orgId, orgName, onClose }: InviteDialogProps) {
  const [email, setEmail] = useState("")
  const [role, setRole] = useState<"member" | "admin">("member")
  const [error, setError] = useState<string | null>(null)
  const [copiedId, setCopiedId] = useState<string | null>(null)
  const backdropRef = useRef<HTMLDivElement>(null)
  const qc = useQueryClient()

  const { data: invites = [] } = useQuery({
    queryKey: ["invites", orgId],
    queryFn: () => inviteApi.list(orgId),
  })

  const create = useMutation({
    mutationFn: () => inviteApi.create(orgId, email, role),
    onSuccess: () => {
      setEmail("")
      setError(null)
      void qc.invalidateQueries({ queryKey: ["invites", orgId] })
    },
    onError: (err) => {
      setError(err instanceof ApiError ? err.message : "Failed to send invite")
    },
  })

  const remove = useMutation({
    mutationFn: (inviteId: string) => inviteApi.remove(orgId, inviteId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["invites", orgId] }),
  })

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "Escape") onClose()
    }
    document.addEventListener("keydown", onKey)
    return () => document.removeEventListener("keydown", onKey)
  }, [onClose])

  async function handleCopy(url: string, id: string) {
    await navigator.clipboard.writeText(url)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const pending = invites.filter((i) => !i.accepted_at)

  return (
    <div
      ref={backdropRef}
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4"
      onClick={(e) => {
        if (e.target === backdropRef.current) onClose()
      }}
    >
      <div className="w-full max-w-md rounded-lg border bg-card shadow-lg">
        <div className="flex items-center justify-between border-b px-4 py-3">
          <div>
            <p className="text-sm font-medium">Invite to {orgName}</p>
            <p className="text-xs text-muted-foreground">
              Invites expire in 7 days
            </p>
          </div>
          <button onClick={onClose} className="rounded p-1 hover:bg-accent">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4 p-4">
          <div className="flex gap-2">
            <Input
              type="email"
              placeholder="colleague@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && email) create.mutate()
              }}
              className="flex-1"
            />
            <select
              value={role}
              onChange={(e) => setRole(e.target.value as "member" | "admin")}
              className="cursor-pointer rounded-md border bg-background px-2 text-sm"
            >
              <option value="member">member</option>
              <option value="admin">admin</option>
            </select>
            <Button
              size="sm"
              disabled={!email || create.isPending}
              onClick={() => create.mutate()}
            >
              {create.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <UserPlus className="h-4 w-4" />
              )}
            </Button>
          </div>

          {error && <p className="text-xs text-destructive">{error}</p>}

          {create.data && (
            <div className="space-y-1.5 rounded-md border border-dashed bg-muted/40 p-3">
              <p className="text-xs font-medium">
                Invite link created — share it once:
              </p>
              <div className="flex items-center gap-2">
                <code className="flex-1 truncate text-[11px] text-muted-foreground">
                  {create.data.invite_url}
                </code>
                <button
                  onClick={() => void handleCopy(create.data.invite_url, "new")}
                  className="shrink-0 rounded p-1 hover:bg-accent"
                  title="Copy"
                >
                  <Copy className="h-3.5 w-3.5" />
                </button>
              </div>
              {copiedId === "new" && (
                <p className="text-[11px] text-muted-foreground">Copied!</p>
              )}
            </div>
          )}

          {pending.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">
                Pending invites
              </p>
              {pending.map((inv) => (
                <div
                  key={inv.id}
                  className="flex items-center gap-2 rounded-md px-2 py-1.5 text-sm hover:bg-accent/50"
                >
                  <span className="flex-1 truncate text-xs">{inv.email}</span>
                  <span
                    className={cn(
                      "rounded px-1.5 py-0.5 text-[10px] font-medium",
                      inv.role === "admin"
                        ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400"
                        : "bg-muted text-muted-foreground"
                    )}
                  >
                    {inv.role}
                  </span>
                  <button
                    onClick={() => remove.mutate(inv.id)}
                    className="shrink-0 rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                    title="Cancel invite"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
