import { create } from 'zustand'

export type ThemePreference = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

export const THEME_STORAGE_KEY = 'portr-admin-theme'

const isPreference = (value: unknown): value is ThemePreference =>
  value === 'system' || value === 'light' || value === 'dark'

export const readStoredPreference = (): ThemePreference => {
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY)
    return isPreference(stored) ? stored : 'system'
  } catch {
    return 'system'
  }
}

const systemQuery = () =>
  typeof window !== 'undefined' && typeof window.matchMedia === 'function'
    ? window.matchMedia('(prefers-color-scheme: dark)')
    : null

export const resolveTheme = (preference: ThemePreference): ResolvedTheme => {
  if (preference !== 'system') return preference
  return systemQuery()?.matches ? 'dark' : 'light'
}

const applyTheme = (resolved: ResolvedTheme) => {
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
}

interface ThemeStore {
  preference: ThemePreference
  resolved: ResolvedTheme
  setPreference: (preference: ThemePreference) => void
}

const initialPreference = readStoredPreference()

export const useThemeStore = create<ThemeStore>((set) => ({
  preference: initialPreference,
  resolved: resolveTheme(initialPreference),
  setPreference: (preference) => {
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, preference)
    } catch {
      // A blocked storage write should not stop the theme from changing.
    }
    const resolved = resolveTheme(preference)
    applyTheme(resolved)
    set({ preference, resolved })
  },
}))

/** Keeps the document in step with the OS while the preference is "system". */
export const startThemeSync = () => {
  applyTheme(useThemeStore.getState().resolved)

  const query = systemQuery()
  if (!query) return () => {}

  const onSystemChange = () => {
    if (useThemeStore.getState().preference !== 'system') return
    const resolved = resolveTheme('system')
    applyTheme(resolved)
    useThemeStore.setState({ resolved })
  }

  query.addEventListener('change', onSystemChange)
  return () => query.removeEventListener('change', onSystemChange)
}
