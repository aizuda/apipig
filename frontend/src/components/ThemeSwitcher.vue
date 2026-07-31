<script setup lang="ts">
import { Button, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@tabtab/ui'
import { useThemeStore, type ThemeMode } from '@/stores/theme'
import { Sun, Moon, Monitor } from '@lucide/vue'

/**
 * 主题切换组件
 * 支持浅色、深色、跟随系统三种主题模式。
 */
const themeStore = useThemeStore()

const themeOptions: { value: ThemeMode; label: string; icon: string }[] = [
  { value: 'light', label: '浅色', icon: 'sun' },
  { value: 'dark', label: '深色', icon: 'moon' },
  { value: 'system', label: '跟随系统', icon: 'monitor' },
]

function getThemeIcon(mode: ThemeMode): string {
  const option = themeOptions.find((o) => o.value === mode)
  return option?.icon || 'monitor'
}

function getThemeLabel(mode: ThemeMode): string {
  const option = themeOptions.find((o) => o.value === mode)
  return option?.label || '跟随系统'
}

function handleToggle(event: MouseEvent) {
  const appliedTheme = themeStore.getAppliedTheme()

  if (themeStore.themeMode === 'system') {
    themeStore.setThemeMode(appliedTheme === 'dark' ? 'light' : 'dark')
  } else if (themeStore.themeMode === 'light' || themeStore.themeMode === 'dark') {
    themeStore.toggleThemeMode(event)
  }
}
</script>

<template>
  <TooltipProvider>
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          variant="ghost"
          size="icon"
          :aria-label="`切换主题，当前为${getThemeLabel(themeStore.themeMode)}模式`"
          @click="handleToggle"
        >
          <Transition name="theme-icon" mode="out-in">
            <Sun
              v-if="getThemeIcon(themeStore.themeMode) === 'sun'"
              :size="20"
              class="text-amber-500"
            />
            <Moon
              v-else-if="getThemeIcon(themeStore.themeMode) === 'moon'"
              :size="20"
              class="text-indigo-400"
            />
            <Monitor v-else :size="20" />
          </Transition>
          <span class="sr-only">切换主题</span>
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <p>当前：{{ getThemeLabel(themeStore.themeMode) }}模式，点击切换</p>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>
</template>

<style scoped>
.theme-icon-enter-active,
.theme-icon-leave-active {
  transition: all 0.2s ease;
}

.theme-icon-enter-from {
  opacity: 0;
  transform: rotate(-90deg) scale(0.5);
}

.theme-icon-leave-to {
  opacity: 0;
  transform: rotate(90deg) scale(0.5);
}
</style>
