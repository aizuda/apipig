<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ErrorPage from './ErrorPage.vue'

const { t } = useI18n()
</script>

<template>
  <ErrorPage
    code="404"
    :title="t('error.pageNotFound')"
    :description="t('error.pageNotFoundDesc')"
    emoji="🔍"
  />
</template>
