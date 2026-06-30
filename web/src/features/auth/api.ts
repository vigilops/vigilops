import { apiClient } from "@/lib/api-client"
import type {
  APIKey,
  AuthUser,
  CreatedAPIKey,
  CreatedInvite,
  InviteInfo,
  OrgMember,
  Organization,
  OrganizationInvite,
  Project,
} from "./types"

export interface LoginPayload {
  email: string
  password: string
}

export interface RegisterPayload {
  email: string
  password: string
  name?: string
}

export const authApi = {
  me: () => apiClient.get<AuthUser>("/v1/auth/me"),
  login: (payload: LoginPayload) =>
    apiClient.post<AuthUser>("/v1/auth/login", payload),
  register: (payload: RegisterPayload) =>
    apiClient.post<AuthUser>("/v1/auth/register", payload),
  logout: () => apiClient.post<void>("/v1/auth/logout"),
  verifyEmail: (token: string) =>
    apiClient.get<void>(`/v1/auth/verify-email/${token}`),
}

export const orgApi = {
  create: (name: string) =>
    apiClient.post<Organization>("/v1/admin/orgs", { name }),
  update: (orgId: string, name: string) =>
    apiClient.patch<Organization>(`/v1/admin/orgs/${orgId}`, { name }),
  delete: (orgId: string) => apiClient.delete(`/v1/admin/orgs/${orgId}`),
}

export const memberApi = {
  list: (orgId: string) =>
    apiClient.get<OrgMember[]>(`/v1/admin/orgs/${orgId}/members`),
  updateRole: (orgId: string, userId: string, role: string) =>
    apiClient.patch<void>(`/v1/admin/orgs/${orgId}/members/${userId}/role`, {
      role,
    }),
  remove: (orgId: string, userId: string) =>
    apiClient.delete(`/v1/admin/orgs/${orgId}/members/${userId}`),
}

export const projectApi = {
  create: (orgId: string, name: string) =>
    apiClient.post<Project>(`/v1/admin/orgs/${orgId}/projects`, { name }),
  list: (orgId: string) =>
    apiClient.get<Project[]>(`/v1/admin/orgs/${orgId}/projects`),
  get: (orgId: string, id: string) =>
    apiClient.get<Project>(`/v1/admin/orgs/${orgId}/projects/${id}`),
  delete: (orgId: string, id: string) =>
    apiClient.delete(`/v1/admin/orgs/${orgId}/projects/${id}`),
}

export const keyApi = {
  create: (orgId: string, projectId: string, name: string) =>
    apiClient.post<CreatedAPIKey>(
      `/v1/admin/orgs/${orgId}/projects/${projectId}/keys`,
      { name }
    ),
  list: (orgId: string, projectId: string) =>
    apiClient.get<APIKey[]>(
      `/v1/admin/orgs/${orgId}/projects/${projectId}/keys`
    ),
  delete: (orgId: string, projectId: string, keyId: string) =>
    apiClient.delete(
      `/v1/admin/orgs/${orgId}/projects/${projectId}/keys/${keyId}`
    ),
}

export const inviteApi = {
  create: (orgId: string, email: string, role = "member") =>
    apiClient.post<CreatedInvite>(`/v1/admin/orgs/${orgId}/invites`, {
      email,
      role,
    }),
  list: (orgId: string) =>
    apiClient.get<OrganizationInvite[]>(`/v1/admin/orgs/${orgId}/invites`),
  remove: (orgId: string, inviteId: string) =>
    apiClient.delete(`/v1/admin/orgs/${orgId}/invites/${inviteId}`),
  getByToken: (token: string) =>
    apiClient.get<InviteInfo>(`/v1/auth/invites/${token}`),
  accept: (token: string) =>
    apiClient.put<void>(`/v1/auth/invites/${token}/accept`),
}
