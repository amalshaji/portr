import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const setSystemDark = (matches: boolean) => {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      matches,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  )
}

/** This jsdom build ships without localStorage, so the test supplies one. */
const store = new Map<string, string>()
const fakeStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, value: string) => void store.set(key, value),
  removeItem: (key: string) => void store.delete(key),
  clear: () => store.clear(),
  key: (index: number) => [...store.keys()][index] ?? null,
  get length() {
    return store.size
  },
} satisfies Storage

/** The store snapshots the stored preference at module load, so each case needs
 *  a fresh module. */
async function loadTheme() {
  vi.resetModules()
  return import('./theme')
}

beforeEach(() => {
  store.clear()
  vi.stubGlobal('localStorage', fakeStorage)
  document.documentElement.className = ''
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('theme store', () => {
  it('follows the system setting when no preference is stored', async () => {
    setSystemDark(true)
    const { useThemeStore } = await loadTheme()

    expect(useThemeStore.getState().preference).toBe('system')
    expect(useThemeStore.getState().resolved).toBe('dark')
  })

  it('lets an explicit choice override the system setting', async () => {
    setSystemDark(true)
    const { useThemeStore } = await loadTheme()

    useThemeStore.getState().setPreference('light')

    expect(useThemeStore.getState().resolved).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('persists the choice across reloads', async () => {
    setSystemDark(false)
    const first = await loadTheme()
    first.useThemeStore.getState().setPreference('dark')
    expect(fakeStorage.getItem('portr-admin-theme')).toBe('dark')

    const second = await loadTheme()
    expect(second.useThemeStore.getState().preference).toBe('dark')
    expect(second.useThemeStore.getState().resolved).toBe('dark')
  })

  it('ignores a stored value that is not a preference', async () => {
    setSystemDark(false)
    fakeStorage.setItem('portr-admin-theme', 'sepia')
    const { useThemeStore } = await loadTheme()

    expect(useThemeStore.getState().preference).toBe('system')
    expect(useThemeStore.getState().resolved).toBe('light')
  })
})
