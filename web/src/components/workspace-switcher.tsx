import { Check, ChevronsUpDown, FolderKanban, Plus } from "lucide-react"
import { useEffect, useRef, useState } from "react"

import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useCreateOrg } from "@/features/auth/hooks/use-auth"
import type { Organization, Project } from "@/features/auth/types"
import { cn } from "@/lib/utils"

export function OrgSwitcher({
  orgs,
  currentOrg,
  onSelect,
}: {
  orgs: Organization[]
  currentOrg: Organization | null
  onSelect: (id: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState("")
  const ref = useRef<HTMLDivElement>(null)
  const createOrg = useCreateOrg()

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setCreating(false)
        setNewName("")
      }
    }
    document.addEventListener("mousedown", onClickOutside)
    return () => document.removeEventListener("mousedown", onClickOutside)
  }, [])

  if (!currentOrg) return null

  function handleCreate() {
    const trimmed = newName.trim()
    if (!trimmed) return
    createOrg.mutate(trimmed, {
      onSuccess: (data) => {
        onSelect(data.id)
        setCreating(false)
        setNewName("")
        setOpen(false)
      },
    })
  }

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 items-center gap-2 rounded-lg border bg-background px-3 text-sm font-medium transition-colors hover:bg-accent"
      >
        <OrgAvatar name={currentOrg.name} />
        <span className="max-w-[140px] truncate">{currentOrg.name}</span>
        <ChevronsUpDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      </button>

      {open && (
        <div className="absolute top-full left-0 z-50 mt-2 min-w-[200px] overflow-hidden rounded-md border bg-popover shadow-md">
          <p className="px-2 py-1.5 text-[11px] font-medium text-muted-foreground">
            Organizations
          </p>
          {orgs.map((org) => (
            <button
              key={org.id}
              onClick={() => {
                onSelect(org.id)
                setOpen(false)
              }}
              className={cn(
                "flex w-full items-center gap-2 px-2 py-2 text-sm transition-colors hover:bg-accent",
                org.id === currentOrg.id && "bg-accent/50"
              )}
            >
              <OrgAvatar name={org.name} />
              <span className="flex-1 truncate text-left">{org.name}</span>
              {org.id === currentOrg.id && (
                <Check className="h-3.5 w-3.5 shrink-0" />
              )}
            </button>
          ))}

          <div className="border-t">
            {creating ? (
              <div className="flex items-center gap-1.5 p-2">
                <Input
                  autoFocus
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleCreate()
                    if (e.key === "Escape") {
                      setCreating(false)
                      setNewName("")
                    }
                  }}
                  placeholder="Organization name"
                  className="h-7 text-xs"
                />
                <Button
                  size="sm"
                  className="h-7 shrink-0 px-2 text-xs"
                  disabled={!newName.trim() || createOrg.isPending}
                  onClick={handleCreate}
                >
                  Create
                </Button>
              </div>
            ) : (
              <button
                onClick={() => setCreating(true)}
                className="flex w-full items-center gap-2 px-2 py-2 text-xs text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
              >
                <Plus className="h-3.5 w-3.5" />
                New organization
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

export function ProjectSwitcher({
  projects,
  currentProject,
  onSelect,
}: {
  projects: Project[]
  currentProject: Project | null
  onSelect: (id: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener("mousedown", onClickOutside)
    return () => document.removeEventListener("mousedown", onClickOutside)
  }, [])

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex h-9 items-center gap-2 rounded-lg border bg-background px-3 text-sm transition-colors hover:bg-accent"
      >
        <FolderKanban className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span
          className={cn(
            "max-w-[140px] truncate",
            !currentProject && "text-muted-foreground"
          )}
        >
          {currentProject ? currentProject.name : "No project"}
        </span>
        <ChevronsUpDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
      </button>

      {open && (
        <div className="absolute top-full left-0 z-50 mt-2 min-w-[200px] overflow-hidden rounded-md border bg-popover shadow-md">
          {projects.length > 0 ? (
            <>
              <p className="px-2 py-1.5 text-[11px] font-medium text-muted-foreground">
                Projects
              </p>
              {projects.map((p) => (
                <button
                  key={p.id}
                  onClick={() => {
                    onSelect(p.id)
                    setOpen(false)
                  }}
                  className={cn(
                    "flex w-full items-center gap-2 px-2 py-2 text-sm transition-colors hover:bg-accent",
                    currentProject?.id === p.id && "bg-accent/50"
                  )}
                >
                  <span className="flex-1 truncate text-left">{p.name}</span>
                  {currentProject?.id === p.id && (
                    <Check className="h-3.5 w-3.5 shrink-0" />
                  )}
                </button>
              ))}
            </>
          ) : (
            <p className="px-2 py-3 text-center text-xs text-muted-foreground">
              No projects
            </p>
          )}
        </div>
      )}
    </div>
  )
}

export function OrgAvatar({ name }: { name: string }) {
  return (
    <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded bg-primary/10 text-[11px] font-semibold text-primary">
      {name.charAt(0).toUpperCase()}
    </span>
  )
}
