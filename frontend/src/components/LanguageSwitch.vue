<script setup lang="ts">
import { Check, Globe, Loader2 } from '@lucide/vue'
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { toast } from '@tabtab/ui'
import { Button } from '@tabtab/ui'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@tabtab/ui'
import { useLocaleStore } from '@/stores/locale'

const { t } = useI18n()

/**
 * 璇█鍒囨崲缁勪欢
 * 鏀寔涓嬫媺閫夋嫨鍜岀洿鎺ュ垏鎹袱绉嶆ā寮? * 鏀寔鍔犺浇鐘舵€佹樉绀? */

const props = withDefaults(
  defineProps<{
    /** 鏄剧ず妯″紡锛歞ropdown-涓嬫媺鑿滃崟, toggle-鍒囨崲鎸夐挳 */
    mode?: 'dropdown' | 'toggle'
    /** 鎸夐挳灏哄 */
    size?: 'default' | 'sm' | 'lg' | 'icon'
    /** 鎸夐挳鍙樹綋 */
    variant?: 'default' | 'secondary' | 'outline' | 'ghost'
  }>(),
  {
    mode: 'dropdown',
    size: 'icon',
    variant: 'ghost',
  },
)

const localeStore = useLocaleStore()

/**
 * 褰撳墠璇█鏄剧ず鏂囨湰
 */
const currentLocaleLabel = computed(() => {
  return localeStore.currentLocaleName
})

/**
 * 鏄惁姝ｅ湪鍔犺浇
 */
const isLoading = computed(() => localeStore.isLoading)

/**
 * 鍒囨崲璇█锛坱oggle 妯″紡锛? */
async function handleToggle() {
  if (props.mode === 'toggle' && !isLoading.value) {
    const newLocale = await localeStore.toggleLocale()
    if (newLocale) {
      toast.success(t('common.switchLanguageSuccess', { name: localeStore.currentLocaleName }))
    } else {
      toast.error(t('common.switchLanguageFailed'))
    }
  }
}

/**
 * 閫夋嫨璇█锛坉ropdown 妯″紡锛? */
async function handleSelect(locale: string) {
  const success = await localeStore.changeLocale(locale as 'zh-CN' | 'en-US')
  if (success) {
    toast.success(t('common.switchLanguageSuccess', { name: localeStore.currentLocaleName }))
  } else {
    toast.error(localeStore.error || t('common.switchLanguageFailed'))
  }
}
</script>

<template>
  <!-- 涓嬫媺鑿滃崟妯″紡 -->
  <DropdownMenu v-if="mode === 'dropdown'">
    <DropdownMenuTrigger as-child>
      <Button :variant="variant" :size="size" class="gap-2" :disabled="isLoading">
        <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
        <Globe v-else class="h-4 w-4" />
        <span v-if="size !== 'icon'" class="hidden sm:inline">{{ currentLocaleLabel }}</span>
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" class="w-40">
      <DropdownMenuItem
        v-for="locale in localeStore.availableLocales"
        :key="locale.value"
        class="cursor-pointer justify-between"
        :disabled="isLoading"
        @select="handleSelect(locale.value)"
      >
        <span>{{ locale.label }}</span>
        <Check v-if="localeStore.currentLocale === locale.value" class="h-4 w-4" />
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <!-- 鍒囨崲鎸夐挳妯″紡 -->
  <Button
    v-else
    :variant="variant"
    :size="size"
    class="gap-2"
    :disabled="isLoading"
    @click="handleToggle"
  >
    <Loader2 v-if="isLoading" class="h-4 w-4 animate-spin" />
    <Globe v-else class="h-4 w-4" />
    <span v-if="size !== 'icon'" class="hidden sm:inline">{{ currentLocaleLabel }}</span>
  </Button>
</template>
