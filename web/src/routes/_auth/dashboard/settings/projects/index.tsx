import { createFileRoute } from "@tanstack/react-router"
import { ChevronDown, ChevronRight, Copy, Plus, Trash2 } from "lucide-react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useAuth, useCurrentProject } from "@/features/auth/hooks/use-auth"
import {
  useKeys,
  useCreateKey,
  useDeleteKey,
} from "@/features/auth/hooks/use-keys"
import { useMyOrgRole } from "@/features/auth/hooks/use-members"
import {
  useCreateProject,
  useDeleteProject,
  useProjects,
} from "@/features/auth/hooks/use-projects"
import type { CreatedAPIKey, Project } from "@/features/auth/types"
import { ApiError } from "@/lib/api-client"

export const Route = createFileRoute("/_auth/dashboard/settings/projects/")({
  component: ProjectsSettingsPage,
})

function ProjectsSettingsPage() {
  const { currentOrg } = useAuth()
  const orgId = currentOrg?.id ?? ""
  const { data: projects = [], isLoading } = useProjects(orgId)
  const { currentProjectId, setCurrentProject } = useCurrentProject()
  const createProject = useCreateProject(orgId)
  const deleteProject = useDeleteProject(orgId)
  const role = useMyOrgRole(orgId)

  const isAdmin = role === "admin" || role === "owner"

  const [newName, setNewName] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<string | null>(null)

  async function handleCreate() {
    const trimmed = newName.trim()
    if (!trimmed) return
    try {
      setCreateError(null)
      const p = await createProject.mutateAsync(trimmed)
      setNewName("")
      setExpanded(p.id)
    } catch (e) {
      setCreateError(
        e instanceof ApiError ? e.message : "Failed to create project"
      )
    }
  }

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div>
        <h2 className="text-sm font-semibold">Projects &amp; API Keys</h2>
        <p className="mt-0.5 text-xs text-muted-foreground">
          Each project has its own API keys and data scope.
        </p>
      </div>

      {isAdmin && (
        <div className="flex flex-col gap-2">
          <div className="flex gap-2">
            <Input
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleCreate()
              }}
              placeholder="New project name"
              className="max-w-64"
            />
            <Button
              size="sm"
              disabled={!newName.trim() || createProject.isPending}
              onClick={() => void handleCreate()}
            >
              <Plus className="h-4 w-4" />
              Create
            </Button>
          </div>
          {createError && (
            <p className="text-xs text-destructive">{createError}</p>
          )}
        </div>
      )}

      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : projects.length === 0 ? (
        <p className="text-sm text-muted-foreground">No projects yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {projects.map((p) => (
            <ProjectRow
              key={p.id}
              project={p}
              orgId={orgId}
              isActive={p.id === currentProjectId}
              isAdmin={isAdmin}
              expanded={expanded === p.id}
              onToggle={() => setExpanded(expanded === p.id ? null : p.id)}
              onSelect={() => setCurrentProject(p.id)}
              onDelete={(id) => deleteProject.mutate(id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function ProjectRow({
  project,
  orgId,
  isActive,
  isAdmin,
  expanded,
  onToggle,
  onSelect,
  onDelete,
}: {
  project: Project
  orgId: string
  isActive: boolean
  isAdmin: boolean
  expanded: boolean
  onToggle: () => void
  onSelect: () => void
  onDelete: (id: string) => void
}) {
  const [confirming, setConfirming] = useState(false)
  const [confirm, setConfirm] = useState("")

  function startDelete() {
    setConfirming(true)
    setConfirm("")
  }

  function cancelDelete() {
    setConfirming(false)
    setConfirm("")
  }

  return (
    <div className="rounded-lg border">
      <div className="flex items-center gap-2 px-3 py-3">
        <button
          onClick={onToggle}
          className="shrink-0 text-muted-foreground hover:text-foreground"
        >
          {expanded ? (
            <ChevronDown className="h-4 w-4" />
          ) : (
            <ChevronRight className="h-4 w-4" />
          )}
        </button>

        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium">{project.name}</p>
          <p className="font-mono text-[11px] text-muted-foreground">
            {project.id}
          </p>
        </div>

        {isActive ? (
          <span className="shrink-0 rounded-full bg-primary/10 px-2 py-0.5 text-[10px] font-medium text-primary">
            active
          </span>
        ) : (
          <Button
            size="sm"
            variant="ghost"
            className="h-7 text-xs"
            onClick={onSelect}
          >
            Switch
          </Button>
        )}

        {isAdmin && !confirming && (
          <button
            onClick={startDelete}
            className="shrink-0 rounded p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
            title="Delete project"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        )}
      </div>

      {confirming && (
        <div className="flex flex-col gap-2 border-t bg-destructive/5 px-4 py-3">
          <p className="text-xs text-muted-foreground">
            Type{" "}
            <span className="font-mono font-medium text-foreground">
              {project.name}
            </span>{" "}
            to confirm deletion. All API keys and data will be permanently
            removed.
          </p>
          <div className="flex items-center gap-2">
            <Input
              autoFocus
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Escape") cancelDelete()
                if (e.key === "Enter" && confirm === project.name)
                  onDelete(project.id)
              }}
              placeholder={project.name}
              className="h-7 max-w-52 text-xs"
            />
            <Button
              size="sm"
              variant="destructive"
              className="h-7 text-xs"
              disabled={confirm !== project.name}
              onClick={() => onDelete(project.id)}
            >
              Delete
            </Button>
            <button
              onClick={cancelDelete}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Cancel
            </button>
          </div>
        </div>
      )}

      {expanded && (
        <div className="border-t bg-muted/20 px-4 py-4">
          <KeysSection orgId={orgId} projectId={project.id} isAdmin={isAdmin} />
        </div>
      )}
    </div>
  )
}

function KeysSection({
  orgId,
  projectId,
  isAdmin,
}: {
  orgId: string
  projectId: string
  isAdmin: boolean
}) {
  const { data: keys = [], isLoading } = useKeys(orgId, projectId)
  const createKey = useCreateKey(orgId, projectId)
  const deleteKey = useDeleteKey(orgId, projectId)

  const [newKeyName, setNewKeyName] = useState("")
  const [createError, setCreateError] = useState<string | null>(null)
  const [freshKey, setFreshKey] = useState<CreatedAPIKey | null>(null)
  const [copied, setCopied] = useState(false)

  async function handleCreate() {
    const trimmed = newKeyName.trim()
    if (!trimmed) return
    try {
      setCreateError(null)
      setFreshKey(null)
      const k = await createKey.mutateAsync(trimmed)
      setFreshKey(k)
      setNewKeyName("")
    } catch (e) {
      setCreateError(e instanceof ApiError ? e.message : "Failed to create key")
    }
  }

  async function handleCopy() {
    if (!freshKey) return
    await navigator.clipboard.writeText(freshKey.key)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-[11px] font-medium tracking-wide text-muted-foreground uppercase">
        API Keys
      </p>

      {isLoading ? (
        <p className="text-xs text-muted-foreground">Loading…</p>
      ) : keys.length === 0 ? (
        <p className="text-xs text-muted-foreground">No keys yet.</p>
      ) : (
        <div className="flex flex-col gap-1">
          {keys.map((k) => (
            <div
              key={k.id}
              className="flex items-center gap-2 rounded px-2 py-1.5 hover:bg-background"
            >
              <span className="flex-1 text-xs font-medium">{k.name}</span>
              <span className="text-[11px] text-muted-foreground">
                {k.last_used_at
                  ? `used ${new Date(k.last_used_at).toLocaleDateString()}`
                  : "never used"}
              </span>
              {isAdmin && (
                <button
                  onClick={() => deleteKey.mutate(k.id)}
                  className="shrink-0 rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                  title="Delete key"
                >
                  <Trash2 className="h-3 w-3" />
                </button>
              )}
            </div>
          ))}
        </div>
      )}

      {freshKey && (
        <div className="space-y-1.5 rounded-md border border-dashed bg-muted/40 p-3">
          <p className="text-xs font-medium">
            Key created — copy it now, it won't be shown again:
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 truncate text-[11px] text-muted-foreground">
              {freshKey.key}
            </code>
            <button
              onClick={() => void handleCopy()}
              className="shrink-0 rounded p-1 hover:bg-accent"
              title="Copy"
            >
              <Copy className="h-3.5 w-3.5" />
            </button>
          </div>
          {copied && (
            <p className="text-[11px] text-muted-foreground">Copied!</p>
          )}
        </div>
      )}

      {isAdmin && (
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center gap-2">
            <Input
              value={newKeyName}
              onChange={(e) => setNewKeyName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") void handleCreate()
              }}
              placeholder="Key name (e.g. production)"
              className="h-7 max-w-52 text-xs"
            />
            <Button
              size="sm"
              variant="outline"
              className="h-7 text-xs"
              disabled={!newKeyName.trim() || createKey.isPending}
              onClick={() => void handleCreate()}
            >
              <Plus className="h-3 w-3" />
              Add key
            </Button>
          </div>
          {createError && (
            <p className="text-xs text-destructive">{createError}</p>
          )}
        </div>
      )}
    </div>
  )
}
