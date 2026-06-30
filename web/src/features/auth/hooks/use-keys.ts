import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { keyApi } from "../api"

export const keyQueryOptions = (orgId: string, projectId: string) =>
  queryOptions({
    queryKey: ["orgs", orgId, "projects", projectId, "keys"],
    queryFn: () => keyApi.list(orgId, projectId),
    enabled: !!orgId && !!projectId,
  })

export function useKeys(orgId: string, projectId: string) {
  return useQuery(keyQueryOptions(orgId, projectId))
}

export function useCreateKey(orgId: string, projectId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => keyApi.create(orgId, projectId, name),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: keyQueryOptions(orgId, projectId).queryKey,
      })
    },
  })
}

export function useDeleteKey(orgId: string, projectId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (keyId: string) => keyApi.delete(orgId, projectId, keyId),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: keyQueryOptions(orgId, projectId).queryKey,
      })
    },
  })
}
