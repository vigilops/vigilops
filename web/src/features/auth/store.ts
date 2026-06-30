import { create } from "zustand"
import { persist } from "zustand/middleware"

interface AuthStore {
  currentOrgId: string | null
  setCurrentOrg: (id: string) => void
  currentProjectId: string | null
  setCurrentProject: (id: string) => void
  clearAuth: () => void
}

export const useAuthStore = create<AuthStore>()(
  persist(
    (set) => ({
      currentOrgId: null,
      setCurrentOrg: (id) => set({ currentOrgId: id }),
      currentProjectId: null,
      setCurrentProject: (id) => set({ currentProjectId: id }),
      clearAuth: () => set({ currentOrgId: null, currentProjectId: null }),
    }),
    { name: "keelwave-auth" }
  )
)
