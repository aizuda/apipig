import { defineStore } from 'pinia'
import { watch, computed, type WatchStopHandle } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import type { LayoutMode, LayoutVariant } from '@tabtab/ui'
import { themeColorConfigs, type ThemeColor } from '@/config/theme-colors'

export type ThemeMode = 'light' | 'dark' | 'system'

export const SIDEBAR_WIDTH_MIN = 184
export const SIDEBAR_WIDTH_MAX = 320
export const SIDEBAR_WIDTH_DEFAULT = 232
export const DEFAULT_THEME_COLOR: ThemeColor = 'blue'

interface ThemeColors {
  primary: string
  primaryForeground: string
  secondary: string
  secondaryForeground: string
  accent: string
  accentForeground: string
  background: string
  foreground: string
  card: string
  cardForeground: string
  popover: string
  popoverForeground: string
  muted: string
  mutedForeground: string
  border: string
  input: string
  ring: string
  destructive: string
  destructiveForeground: string
  sidebar: string
  sidebarForeground: string
  sidebarPrimary: string
  sidebarPrimaryForeground: string
  sidebarAccent: string
  sidebarAccentForeground: string
  sidebarBorder: string
  sidebarRing: string
}

function extractHue(oklchColor: string): number {
  const match = oklchColor.match(/oklch\([\d.]+\s+[\d.]+\s+([\d.]+)\)/)
  return match ? parseFloat(match[1]) : 0
}

const baseLightColors: Partial<ThemeColors> = {
  secondary: 'oklch(0.97 0 0)',
  secondaryForeground: 'oklch(0.205 0 0)',
  background: 'oklch(1 0 0)',
  foreground: 'oklch(0.145 0 0)',
  card: 'oklch(1 0 0)',
  cardForeground: 'oklch(0.145 0 0)',
  popover: 'oklch(1 0 0)',
  popoverForeground: 'oklch(0.145 0 0)',
  muted: 'oklch(0.97 0 0)',
  border: 'oklch(0.922 0 0)',
  input: 'oklch(0.922 0 0)',
  destructive: 'oklch(0.577 0.245 27.325)',
  destructiveForeground: 'oklch(0.985 0 0)',
  sidebar: 'oklch(0.985 0 0)',
  sidebarForeground: 'oklch(0.145 0 0)',
  sidebarAccent: 'oklch(0.97 0 0)',
  sidebarAccentForeground: 'oklch(0.205 0 0)',
  sidebarBorder: 'oklch(0.922 0 0)',
}

const baseDarkColors: Partial<ThemeColors> = {
  secondary: 'oklch(0.269 0 0)',
  secondaryForeground: 'oklch(0.985 0 0)',
  background: 'oklch(0.145 0 0)',
  foreground: 'oklch(0.985 0 0)',
  card: 'oklch(0.205 0 0)',
  cardForeground: 'oklch(0.985 0 0)',
  popover: 'oklch(0.205 0 0)',
  popoverForeground: 'oklch(0.985 0 0)',
  muted: 'oklch(0.269 0 0)',
  border: 'oklch(1 0 0 / 10%)',
  input: 'oklch(1 0 0 / 15%)',
  destructive: 'oklch(0.704 0.191 22.216)',
  destructiveForeground: 'oklch(0.985 0 0)',
  sidebar: 'oklch(0.205 0 0)',
  sidebarForeground: 'oklch(0.985 0 0)',
  sidebarAccent: 'oklch(0.269 0 0)',
  sidebarAccentForeground: 'oklch(0.985 0 0)',
  sidebarBorder: 'oklch(1 0 0 / 10%)',
}

function generateThemeColors(config: (typeof themeColorConfigs)[ThemeColor]): {
  light: ThemeColors
  dark: ThemeColors
} {
  const primary = config.primary
  const darkPrimary = config.darkPrimary || primary
  const primaryForeground = config.primaryForeground || 'oklch(0.985 0 0)'
  const darkPrimaryForeground = config.darkPrimaryForeground || primaryForeground
  const accent = config.accent || primary

  const lightDestructive = 'oklch(0.577 0.245 27.325)'
  const darkDestructive = 'oklch(0.704 0.191 22.216)'

  const hue = extractHue(primary)
  const darkHue = extractHue(darkPrimary)

  const lightMutedForeground = `oklch(0.55 0.02 ${hue})`
  const darkMutedForeground = `oklch(0.68 0.03 ${darkHue})`

  return {
    light: {
      ...baseLightColors,
      primary,
      primaryForeground,
      accent: accent,
      accentForeground: primaryForeground,
      ring: primary,
      mutedForeground: lightMutedForeground,
      destructive: lightDestructive,
      destructiveForeground: 'oklch(0.985 0 0)',
      sidebarPrimary: primary,
      sidebarPrimaryForeground: primaryForeground,
      sidebarRing: primary,
      sidebarAccent: accent,
      sidebarAccentForeground: primaryForeground,
    } as ThemeColors,
    dark: {
      ...baseDarkColors,
      primary: darkPrimary,
      primaryForeground: darkPrimaryForeground,
      accent: darkPrimary,
      accentForeground: darkPrimaryForeground,
      ring: darkPrimary,
      mutedForeground: darkMutedForeground,
      destructive: darkDestructive,
      destructiveForeground: 'oklch(0.985 0 0)',
      sidebarPrimary: darkPrimary,
      sidebarPrimaryForeground: darkPrimaryForeground,
      sidebarRing: darkPrimary,
      sidebarAccent: darkPrimary,
      sidebarAccentForeground: darkPrimaryForeground,
    } as ThemeColors,
  }
}

const presetThemes: Record<
  ThemeColor,
  { name: string; colors: { light: ThemeColors; dark: ThemeColors } }
> = Object.fromEntries(
  Object.entries(themeColorConfigs).map(([key, config]) => [
    key,
    { name: config.name, colors: generateThemeColors(config) },
  ]),
) as Record<ThemeColor, { name: string; colors: { light: ThemeColors; dark: ThemeColors } }>

const cssVarMap: Record<keyof ThemeColors, string> = {
  primary: '--primary',
  primaryForeground: '--primary-foreground',
  secondary: '--secondary',
  secondaryForeground: '--secondary-foreground',
  accent: '--accent',
  accentForeground: '--accent-foreground',
  background: '--background',
  foreground: '--foreground',
  card: '--card',
  cardForeground: '--card-foreground',
  popover: '--popover',
  popoverForeground: '--popover-foreground',
  muted: '--muted',
  mutedForeground: '--muted-foreground',
  border: '--border',
  input: '--input',
  ring: '--ring',
  destructive: '--destructive',
  destructiveForeground: '--destructive-foreground',
  sidebar: '--sidebar',
  sidebarForeground: '--sidebar-foreground',
  sidebarPrimary: '--sidebar-primary',
  sidebarPrimaryForeground: '--sidebar-primary-foreground',
  sidebarAccent: '--sidebar-accent',
  sidebarAccentForeground: '--sidebar-accent-foreground',
  sidebarBorder: '--sidebar-border',
  sidebarRing: '--sidebar-ring',
}

export const useThemeStore = defineStore('theme', () => {
  const themeMode = useLocalStorage<ThemeMode>('theme-mode', 'system')
  const themeColor = useLocalStorage<ThemeColor>('theme-color', DEFAULT_THEME_COLOR)
  const borderRadius = useLocalStorage<number>('theme-radius', 0.625)
  const layoutMode = useLocalStorage<LayoutMode>('layout-mode', 'sidebar')
  const layoutVariant = useLocalStorage<LayoutVariant>('layout-variant', 'streamer')
  const sidebarCollapsed = useLocalStorage<boolean>('sidebar-collapsed', false)
  const sidebarWidth = useLocalStorage<number>('sidebar-width', SIDEBAR_WIDTH_DEFAULT)
  const showTabs = useLocalStorage<boolean>('show-tabs', true)
  const tabsFixed = useLocalStorage<boolean>('tabs-fixed', true)

  const cleanupFns: Array<() => void> = []
  const watchStops: WatchStopHandle[] = []

  const normalizedSidebarWidth =
    sidebarWidth.value === 260
      ? SIDEBAR_WIDTH_DEFAULT
      : Math.min(SIDEBAR_WIDTH_MAX, Math.max(SIDEBAR_WIDTH_MIN, sidebarWidth.value))

  if (sidebarWidth.value !== normalizedSidebarWidth) {
    sidebarWidth.value = normalizedSidebarWidth
  }

  function registerCleanup(fn: () => void): void {
    cleanupFns.push(fn)
  }

  function registerWatchStop(stop: WatchStopHandle): void {
    watchStops.push(stop)
  }

  const currentColors = computed(() => {
    const theme = presetThemes[themeColor.value]
    const mode = getAppliedTheme()
    return theme?.colors[mode] || presetThemes[DEFAULT_THEME_COLOR].colors.light
  })

  const availableThemes = computed(() =>
    Object.entries(presetThemes).map(([key, value]) => ({
      key,
      name: value.name,
      primaryColor: value.colors.light.primary,
      darkPrimaryColor: value.colors.dark.primary,
    })),
  )

  function getSystemTheme(): 'light' | 'dark' {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
  }

  function getAppliedTheme(): 'light' | 'dark' {
    if (themeMode.value === 'system') {
      return getSystemTheme()
    }
    return themeMode.value
  }

  function applyTheme() {
    const appliedTheme = getAppliedTheme()
    const html = document.documentElement

    if (appliedTheme === 'dark') {
      html.classList.add('dark')
    } else {
      html.classList.remove('dark')
    }
  }

  function applyThemeColors() {
    const colors = currentColors.value
    if (!colors) return

    const root = document.documentElement

    Object.entries(cssVarMap).forEach(([key, cssVar]) => {
      root.style.setProperty(cssVar, colors[key as keyof ThemeColors])
    })
  }

  function applyLayoutConfig() {
    const root = document.documentElement
    root.style.setProperty('--radius', `${borderRadius.value}rem`)
  }

  function setMode(mode: 'light' | 'dark') {
    themeMode.value = mode
    applyTheme()
    applyThemeColors()
  }

  async function toggleThemeMode(event?: MouseEvent) {
    const currentMode = getAppliedTheme()
    const newMode = currentMode === 'light' ? 'dark' : 'light'

    if ('startViewTransition' in document) {
      const x = event?.clientX ?? window.innerWidth / 2
      const y = event?.clientY ?? window.innerHeight / 2
      const endRadius = Math.hypot(
        Math.max(x, window.innerWidth - x),
        Math.max(y, window.innerHeight - y),
      )

      const transition = (
        document as Document & {
          startViewTransition: (callback: () => void) => { ready: Promise<void> }
        }
      ).startViewTransition(() => setMode(newMode))

      await transition.ready

      document.documentElement.animate(
        [
          { clipPath: `circle(0px at ${x}px ${y}px)` },
          { clipPath: `circle(${endRadius}px at ${x}px ${y}px)` },
        ],
        {
          duration: 500,
          easing: 'ease-in-out',
          pseudoElement: '::view-transition-new(root)',
        },
      )
    } else {
      setMode(newMode)
    }
  }

  function setThemeMode(mode: ThemeMode) {
    themeMode.value = mode
    applyTheme()
    applyThemeColors()
  }

  function setLayoutMode(mode: LayoutMode) {
    layoutMode.value = mode
  }

  function setLayoutVariant(variant: LayoutVariant) {
    layoutVariant.value = variant
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setSidebarCollapsed(collapsed: boolean) {
    sidebarCollapsed.value = collapsed
  }

  function setSidebarWidth(width: number) {
    sidebarWidth.value = Math.min(SIDEBAR_WIDTH_MAX, Math.max(SIDEBAR_WIDTH_MIN, width))
  }

  function setShowTabs(show: boolean) {
    showTabs.value = show
  }

  function toggleShowTabs() {
    showTabs.value = !showTabs.value
  }

  function setTabsFixed(fixed: boolean) {
    tabsFixed.value = fixed
  }

  function toggleTabsFixed() {
    tabsFixed.value = !tabsFixed.value
  }

  function setThemeColor(color: ThemeColor) {
    themeColor.value = color
    applyThemeColors()
  }

  function setBorderRadius(radius: number) {
    borderRadius.value = radius
    applyLayoutConfig()
  }

  function resetThemeSettings() {
    setThemeColor(DEFAULT_THEME_COLOR)
    setBorderRadius(0.625)
    setLayoutVariant('fixed')
    setSidebarCollapsed(false)
    setSidebarWidth(SIDEBAR_WIDTH_DEFAULT)
    setShowTabs(true)
    setTabsFixed(true)
  }

  function cleanup() {
    cleanupFns.forEach((fn) => fn())
    cleanupFns.length = 0
    watchStops.forEach((stop) => stop())
    watchStops.length = 0
  }

  function initTheme() {
    applyTheme()
    applyThemeColors()
    applyLayoutConfig()

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
    const handleChange = () => {
      if (themeMode.value === 'system') {
        applyTheme()
        applyThemeColors()
      }
    }

    mediaQuery.addEventListener('change', handleChange)
    registerCleanup(() => mediaQuery.removeEventListener('change', handleChange))
  }

  const themeModeWatchStop = watch(themeMode, () => {
    applyTheme()
    applyThemeColors()
  })
  registerWatchStop(themeModeWatchStop)

  return {
    themeMode,
    themeColor,
    borderRadius,
    layoutMode,
    layoutVariant,
    sidebarCollapsed,
    sidebarWidth,
    showTabs,
    tabsFixed,
    currentColors,
    availableThemes,
    setThemeMode,
    toggleThemeMode,
    setThemeColor,
    setBorderRadius,
    setLayoutMode,
    setLayoutVariant,
    toggleSidebar,
    setSidebarCollapsed,
    setSidebarWidth,
    setShowTabs,
    toggleShowTabs,
    setTabsFixed,
    toggleTabsFixed,
    initTheme,
    getAppliedTheme,
    resetThemeSettings,
    cleanup,
  }
})
