<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const route = useRoute()
defineOptions({ name: 'SettingsRoleDetail' })
</script>

<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-3xl font-bold">{{ t('settings.roleDetail') }}</h1>
      <p class="text-muted-foreground">{{ t('settings.roleDetailDesc') }}</p>
    </div>
    <div class="text-center text-muted-foreground py-12">
      {{ t('settings.roleDetailPage', { id: route.params.id }) }}
    </div>
  </div>
</template>
