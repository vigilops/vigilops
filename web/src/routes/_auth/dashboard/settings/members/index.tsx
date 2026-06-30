import { createFileRoute } from "@tanstack/react-router"
import { Trash2 } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { InviteDialog } from "@/features/auth/components/invite-dialog"
import { useAuth } from "@/features/auth/hooks/use-auth"
import {
  useMembers,
  useMyOrgRole,
  useRemoveMember,
  useUpdateMemberRole,
} from "@/features/auth/hooks/use-members"

export const Route = createFileRoute("/_auth/dashboard/settings/members/")({
  component: MembersSettingsPage,
})

function MembersSettingsPage() {
  const { currentOrg, user } = useAuth()
  const orgId = currentOrg?.id ?? ""
  const { data: members = [], isLoading } = useMembers(orgId)
  const updateRole = useUpdateMemberRole(orgId)
  const remove = useRemoveMember(orgId)
  const role = useMyOrgRole(orgId)
  const [inviteOpen, setInviteOpen] = useState(false)

  const isOwner = role === "owner"
  const isAdmin = role === "admin" || isOwner

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      {inviteOpen && currentOrg && (
        <InviteDialog
          orgId={currentOrg.id}
          orgName={currentOrg.name}
          onClose={() => setInviteOpen(false)}
        />
      )}

      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-semibold">Members</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {members.length} member{members.length !== 1 ? "s" : ""}
          </p>
        </div>
        {isAdmin && (
          <Button
            size="sm"
            variant="outline"
            onClick={() => setInviteOpen(true)}
          >
            Invite member
          </Button>
        )}
      </div>

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : (
        <div className="divide-y rounded-lg border">
          {members.map((m) => (
            <div key={m.user_id} className="flex items-center gap-3 px-4 py-3">
              <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10 text-xs font-semibold text-primary">
                {(m.name || m.email).charAt(0).toUpperCase()}
              </div>

              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {m.name || m.email}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  {m.email}
                </p>
              </div>

              <RoleCell
                role={m.role}
                canChange={
                  isOwner && m.role !== "owner" && m.user_id !== user?.id
                }
                onChange={(newRole) =>
                  updateRole.mutate({ userId: m.user_id, role: newRole })
                }
              />

              {isAdmin && m.role !== "owner" && m.user_id !== user?.id && (
                <button
                  onClick={() => remove.mutate(m.user_id)}
                  className="shrink-0 rounded p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                  title="Remove member"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function RoleCell({
  role,
  canChange,
  onChange,
}: {
  role: string
  canChange: boolean
  onChange: (role: string) => void
}) {
  if (!canChange) {
    return (
      <span className="rounded bg-muted px-2 py-1 text-xs text-muted-foreground capitalize">
        {role}
      </span>
    )
  }
  return (
    <select
      value={role}
      onChange={(e) => onChange(e.target.value)}
      className="cursor-pointer rounded border bg-background px-2 py-1 text-xs"
    >
      <option value="member">member</option>
      <option value="admin">admin</option>
    </select>
  )
}
