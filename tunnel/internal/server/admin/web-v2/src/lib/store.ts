import { create } from 'zustand'
import type { CurrentTeamUser } from '@/types'

interface UserStore {
  currentUser: CurrentTeamUser | null
  setCurrentUser: (user: CurrentTeamUser | null) => void
}

export const useUserStore = create<UserStore>((set) => ({
  currentUser: null,
  setCurrentUser: (user) => set({ currentUser: user }),
}))
