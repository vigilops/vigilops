import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { projectApi } from "../api"

export const projectQueryOptions = (orgId: string) =>
  queryOptions({
    queryKey: ["orgs", orgId, "projects"],
    queryFn: () => projectApi.list(orgId),
    enabled: !!orgId,
  })

export function useProjects(orgId: string) {
  return useQuery(projectQueryOptions(orgId))
}

export function useCreateProject(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => projectApi.create(orgId, name),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: projectQueryOptions(orgId).queryKey,
      })
    },
  })
}

export function useDeleteProject(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (projectId: string) => projectApi.delete(orgId, projectId),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: projectQueryOptions(orgId).queryKey,
      })
    },
  })
}
