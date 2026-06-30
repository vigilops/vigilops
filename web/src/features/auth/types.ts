export interface User {
  id: string
  email: string
  name: string
  is_verified: boolean
  verified_at: string | null
  created_at: string
}

export interface Organization {
  id: string
  name: string
  created_at: string
}

export interface OrganizationInvite {
  id: string
  organization_id: string
  email: string
  role: string
  expires_at: string
  accepted_at: string | null
  created_at: string
}

export interface InviteInfo {
  org_name: string
  role: string
}

export interface CreatedInvite extends OrganizationInvite {
  invite_url: string
}

export interface AuthUser {
  user: User
  organizations: Organization[]
}

export interface Project {
  id: string
  name: string
  organization_id: string
  created_at: string
}

export interface APIKey {
  id: string
  project_id: string
  name: string
  last_used_at: string | null
  created_at: string
}

export interface CreatedAPIKey extends APIKey {
  key: string
}

export interface OrgMember {
  organization_id: string
  user_id: string
  role: string
  created_at: string
  email: string
  name: string
}
