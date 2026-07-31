import type { Plugin } from 'vite'
import tailwindcss from '@tailwindcss/vite'

export default function tabtabUI(): Plugin[] {
  return [...tailwindcss()]
}
