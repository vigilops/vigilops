import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { useEffect } from "react"
import { authApi, orgApi } from "../api"
import { useAuthStore } from "../store"
import { useProjects } from "./use-projects"
import type { Organization } from "../types"

export function useUpdateOrg() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orgId, name }: { orgId: string; name: string }) =>
      orgApi.update(orgId, name),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meQueryOptions.queryKey })
    },
  })
}

export function useDeleteOrg() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (orgId: string) => orgApi.delete(orgId),
    onSuccess: () => {
      qc.removeQueries({ queryKey: meQueryOptions.queryKey })
    },
  })
}

export const meQueryOptions = queryOptions({
  queryKey: ["auth", "me"],
  queryFn: authApi.me,
  retry: false,
  staleTime: 1000 * 60 * 5,
})

export function useMe() {
  return useQuery(meQueryOptions)
}

/** Single hook for all auth state — server data + persisted client selection. */
export function useAuth() {
  const qc = useQueryClient()
  const { data, isPending, isError } = useMe()
  const currentOrgId = useAuthStore((s) => s.currentOrgId)
  const setCurrentOrg = useAuthStore((s) => s.setCurrentOrg)
  const clearAuth = useAuthStore((s) => s.clearAuth)

  const orgs: Organization[] = data?.organizations ?? []
  const currentOrg: Organization | null =
    orgs.find((o) => o.id === currentOrgId) ?? orgs.at(0) ?? null

  useEffect(() => {
    if (currentOrg && currentOrg.id !== currentOrgId) {
      setCurrentOrg(currentOrg.id)
    }
  }, [currentOrg, currentOrgId, setCurrentOrg])

  function switchOrg(id: string) {
    setCurrentOrg(id)
    qc.clear()
  }

  return {
    user: data?.user ?? null,
    orgs,
    currentOrg,
    isLoading: isPending,
    isAuthenticated: !!data?.user,
    isError,
    setCurrentOrg: switchOrg,
    clearAuth,
  }
}

/** Project selection within the current org. Auto-selects first project. */
export function useCurrentProject() {
  const qc = useQueryClient()
  const { currentOrg } = useAuth()
  const currentProjectId = useAuthStore((s) => s.currentProjectId)
  const setCurrentProject = useAuthStore((s) => s.setCurrentProject)
  const { data: projects, isPending } = useProjects(currentOrg?.id ?? "")

  useEffect(() => {
    if (!projects?.length) return
    const valid = projects.some((p) => p.id === currentProjectId)
    if (!valid) setCurrentProject(projects[0].id)
  }, [projects, currentProjectId, setCurrentProject])

  const currentProject =
    projects?.find((p) => p.id === currentProjectId) ?? projects?.[0] ?? null

  function switchProject(id: string) {
    setCurrentProject(id)
    qc.clear()
  }

  return {
    currentProject,
    currentProjectId: currentProject?.id ?? null,
    isLoadingProject: isPending,
    hasNoProjects: !isPending && projects?.length === 0,
    projects: projects ?? [],
    setCurrentProject: switchProject,
  }
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: authApi.login,
    onSuccess: (data) => {
      qc.setQueryData(meQueryOptions.queryKey, data)
    },
  })
}

export function useRegister() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: authApi.register,
    onSuccess: (data) => {
      qc.setQueryData(meQueryOptions.queryKey, data)
    },
  })
}

export function useCreateOrg() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => orgApi.create(name),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: meQueryOptions.queryKey })
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  const { clearAuth } = useAuthStore()
  return useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      clearAuth()
      qc.removeQueries({ queryKey: ["auth", "me"] })
      qc.clear()
    },
  })
}
