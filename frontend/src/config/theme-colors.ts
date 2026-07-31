export type ThemeColor =
  | 'neutral'
  | 'slate'
  | 'stone'
  | 'red'
  | 'rose'
  | 'orange'
  | 'amber'
  | 'yellow'
  | 'lime'
  | 'green'
  | 'emerald'
  | 'teal'
  | 'cyan'
  | 'sky'
  | 'blue'
  | 'indigo'
  | 'violet'
  | 'purple'
  | 'fuchsia'
  | 'pink'

export interface ThemeColorConfig {
  name: string
  primary: string
  primaryForeground?: string
  darkPrimary?: string
  darkPrimaryForeground?: string
  accent?: string
}

export const themeColorConfigs: Record<ThemeColor, ThemeColorConfig> = {
  neutral: {
    name: '默认',
    primary: 'oklch(0.205 0 0)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.45 0 0)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.708 0 0)',
  },
  slate: {
    name: '岩灰',
    primary: 'oklch(0.55 0.04 260)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.75 0.04 260)',
    accent: 'oklch(0.65 0.05 260)',
  },
  stone: {
    name: '石色',
    primary: 'oklch(0.55 0.02 80)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.72 0.02 80)',
    accent: 'oklch(0.68 0.03 80)',
  },
  red: {
    name: '红色',
    primary: 'oklch(0.55 0.22 25)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.75 0.18 25)',
    accent: 'oklch(0.65 0.22 25)',
  },
  rose: {
    name: '玫瑰',
    primary: 'oklch(0.58 0.18 15)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.76 0.15 15)',
    accent: 'oklch(0.68 0.18 15)',
  },
  orange: {
    name: '橙色',
    primary: 'oklch(0.62 0.18 45)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.78 0.14 45)',
    accent: 'oklch(0.70 0.18 45)',
  },
  amber: {
    name: '琥珀',
    primary: 'oklch(0.72 0.16 75)',
    primaryForeground: 'oklch(0.25 0 0)',
    darkPrimary: 'oklch(0.70 0.14 75)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.75 0.16 75)',
  },
  yellow: {
    name: '黄色',
    primary: 'oklch(0.78 0.17 85)',
    primaryForeground: 'oklch(0.25 0 0)',
    darkPrimary: 'oklch(0.68 0.15 85)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.80 0.15 85)',
  },
  lime: {
    name: '青柠',
    primary: 'oklch(0.72 0.2 125)',
    primaryForeground: 'oklch(0.25 0 0)',
    darkPrimary: 'oklch(0.65 0.18 125)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.75 0.18 125)',
  },
  green: {
    name: '绿色',
    primary: 'oklch(0.55 0.15 145)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.72 0.12 145)',
    accent: 'oklch(0.68 0.14 145)',
  },
  emerald: {
    name: '翠绿',
    primary: 'oklch(0.58 0.16 160)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.74 0.12 160)',
    accent: 'oklch(0.70 0.14 160)',
  },
  teal: {
    name: '青绿',
    primary: 'oklch(0.55 0.12 185)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.72 0.10 185)',
    accent: 'oklch(0.68 0.11 185)',
  },
  cyan: {
    name: '青色',
    primary: 'oklch(0.65 0.14 195)',
    primaryForeground: 'oklch(0.25 0 0)',
    darkPrimary: 'oklch(0.60 0.12 195)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.72 0.12 195)',
  },
  sky: {
    name: '天空',
    primary: 'oklch(0.62 0.13 225)',
    primaryForeground: 'oklch(0.25 0 0)',
    darkPrimary: 'oklch(0.58 0.12 225)',
    darkPrimaryForeground: 'oklch(0.985 0 0)',
    accent: 'oklch(0.70 0.12 225)',
  },
  blue: {
    name: '蓝色',
    primary: 'oklch(0.55 0.2 260)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.72 0.16 260)',
    accent: 'oklch(0.65 0.18 260)',
  },
  indigo: {
    name: '靛蓝',
    primary: 'oklch(0.52 0.18 275)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.70 0.14 275)',
    accent: 'oklch(0.62 0.16 275)',
  },
  violet: {
    name: '紫罗兰',
    primary: 'oklch(0.58 0.2 285)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.75 0.15 285)',
    accent: 'oklch(0.68 0.18 285)',
  },
  purple: {
    name: '紫色',
    primary: 'oklch(0.55 0.22 300)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.72 0.17 300)',
    accent: 'oklch(0.65 0.20 300)',
  },
  fuchsia: {
    name: '洋红',
    primary: 'oklch(0.6 0.22 325)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.76 0.17 325)',
    accent: 'oklch(0.68 0.20 325)',
  },
  pink: {
    name: '粉色',
    primary: 'oklch(0.62 0.2 355)',
    primaryForeground: 'oklch(0.985 0 0)',
    darkPrimary: 'oklch(0.78 0.15 355)',
    accent: 'oklch(0.70 0.18 355)',
  },
}
