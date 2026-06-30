import {
  queryOptions,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query"
import { memberApi } from "../api"
import { useAuth } from "./use-auth"

export const memberQueryOptions = (orgId: string) =>
  queryOptions({
    queryKey: ["orgs", orgId, "members"],
    queryFn: () => memberApi.list(orgId),
    enabled: !!orgId,
  })

export function useMembers(orgId: string) {
  return useQuery(memberQueryOptions(orgId))
}

export function useUpdateMemberRole(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ userId, role }: { userId: string; role: string }) =>
      memberApi.updateRole(orgId, userId, role),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: memberQueryOptions(orgId).queryKey,
      })
    },
  })
}

export function useRemoveMember(orgId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) => memberApi.remove(orgId, userId),
    onSuccess: () => {
      void qc.invalidateQueries({
        queryKey: memberQueryOptions(orgId).queryKey,
      })
    },
  })
}

export function useMyOrgRole(orgId: string): string | null {
  const { user } = useAuth()
  const { data: members } = useMembers(orgId)
  return members?.find((m) => m.user_id === user?.id)?.role ?? null
}
