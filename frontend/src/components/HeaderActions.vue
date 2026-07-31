<script setup lang="ts">
import type { MenuItem } from '@tabtab/ui'
import { LayoutSearch, Button } from '@tabtab/ui'
import { ExternalLink } from '@lucide/vue'
import Notification from '@/components/Notification.vue'
import LanguageSwitch from '@/components/LanguageSwitch.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import ThemeSettings from '@/components/ThemeSettings.vue'

defineProps<{
  visibleMenuItems: MenuItem[]
  placeholder?: string
  description?: string
  emptyText?: string
  groupTitles?: {
    recent?: string
    navigation?: string
    actions?: string
  }
  showThemeSettings?: boolean
}>()

const showThemeSettings = defineModel<boolean>('showThemeSettings', { default: true })
</script>

<template>
  <div class="flex min-w-0 items-center gap-0.5 sm:gap-1.5">
    <Button variant="ghost" size="icon" class="hidden lg:inline-flex" as-child>
      <a href="https://apipig.aizuda.com" target="_blank" rel="noopener noreferrer">
        <ExternalLink :size="20" class="shrink-0" />
      </a>
    </Button>
    <LayoutSearch
      :menu-items="visibleMenuItems"
      :placeholder="placeholder || 'Search...'"
      :description="description"
      :empty-text="emptyText"
      :group-titles="groupTitles"
    />
    <Notification />
    <div class="hidden sm:block">
      <LanguageSwitch mode="dropdown" size="icon" variant="ghost" />
    </div>
    <ThemeSwitcher />
    <div v-if="showThemeSettings" class="hidden md:block">
      <ThemeSettings />
    </div>
  </div>
</template>
