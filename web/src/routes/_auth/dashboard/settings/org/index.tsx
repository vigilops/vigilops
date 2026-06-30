import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  useAuth,
  useDeleteOrg,
  useUpdateOrg,
} from "@/features/auth/hooks/use-auth"
import { useMyOrgRole } from "@/features/auth/hooks/use-members"
import { ApiError } from "@/lib/api-client"

export const Route = createFileRoute("/_auth/dashboard/settings/org/")({
  component: OrgSettingsPage,
})

function OrgSettingsPage() {
  const { currentOrg, clearAuth } = useAuth()
  const navigate = useNavigate()
  const role = useMyOrgRole(currentOrg?.id ?? "")

  const [name, setName] = useState(currentOrg?.name ?? "")
  const [nameError, setNameError] = useState<string | null>(null)

  useEffect(() => {
    setName(currentOrg?.name ?? "")
    setNameError(null)
  }, [currentOrg?.id])
  const [deleteConfirm, setDeleteConfirm] = useState("")
  const [deleteError, setDeleteError] = useState<string | null>(null)

  const updateOrg = useUpdateOrg()
  const deleteOrg = useDeleteOrg()

  if (!currentOrg) return null

  async function handleRename() {
    const trimmed = name.trim()
    if (!trimmed || trimmed === currentOrg.name) return
    try {
      setNameError(null)
      await updateOrg.mutateAsync({ orgId: currentOrg.id, name: trimmed })
    } catch (e) {
      setNameError(e instanceof ApiError ? e.message : "Failed to rename")
    }
  }

  async function handleDelete() {
    if (deleteConfirm !== currentOrg.name) return
    try {
      setDeleteError(null)
      await deleteOrg.mutateAsync(currentOrg.id)
      clearAuth()
      void navigate({ to: "/onboarding" })
    } catch (e) {
      setDeleteError(
        e instanceof ApiError ? e.message : "Failed to delete organization"
      )
    }
  }

  return (
    <div className="flex max-w-xl flex-col gap-8">
      <section className="flex flex-col gap-4">
        <div>
          <h2 className="text-sm font-semibold">Organization name</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Shown across the dashboard and invites.
          </p>
        </div>
        <div className="flex gap-2">
          <Input
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void handleRename()
            }}
            className="max-w-64"
          />
          <Button
            size="sm"
            disabled={
              !name.trim() ||
              name.trim() === currentOrg.name ||
              updateOrg.isPending
            }
            onClick={() => void handleRename()}
          >
            Save
          </Button>
        </div>
        {nameError && <p className="text-xs text-destructive">{nameError}</p>}
        {updateOrg.isSuccess && (
          <p className="text-xs text-muted-foreground">Saved.</p>
        )}
      </section>

      <hr />

      <section className="flex flex-col gap-4">
        <div>
          <h2 className="text-sm font-semibold text-destructive">
            Danger zone
          </h2>
          <p className="mt-0.5 text-xs text-muted-foreground">
            Permanently delete this organization, all projects, API keys, and
            collected data. Cannot be undone.
          </p>
        </div>

        {role === "owner" ? (
          <div className="flex flex-col gap-3 rounded-lg border border-destructive/30 p-4">
            <p className="text-xs text-muted-foreground">
              Type{" "}
              <span className="font-mono font-medium text-foreground">
                {currentOrg.name}
              </span>{" "}
              to confirm deletion.
            </p>
            <Input
              value={deleteConfirm}
              onChange={(e) => setDeleteConfirm(e.target.value)}
              placeholder={currentOrg.name}
              className="max-w-64"
            />
            <Button
              variant="destructive"
              size="sm"
              className="w-fit"
              disabled={
                deleteConfirm !== currentOrg.name || deleteOrg.isPending
              }
              onClick={() => void handleDelete()}
            >
              Delete organization
            </Button>
            {deleteError && (
              <p className="text-xs text-destructive">{deleteError}</p>
            )}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">
            Only the organization owner can delete it.
          </p>
        )}
      </section>
    </div>
  )
}
