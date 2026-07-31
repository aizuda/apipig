<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ErrorPage from './ErrorPage.vue'

const { t } = useI18n()
</script>

<template>
  <ErrorPage
    code="403"
    :title="t('error.accessDenied')"
    :description="t('error.accessDeniedDesc')"
    emoji="🚫"
  >
    <template #hint>
      {{ t('error.contactAdminHint') }}
    </template>
  </ErrorPage>
</template>
