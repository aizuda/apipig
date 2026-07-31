<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { LogIn } from '@lucide/vue'
import ErrorPage from './ErrorPage.vue'
import { Button } from '@tabtab/ui'

const { t } = useI18n()
const router = useRouter()

function goToLogin() {
  router.push({ name: 'Login' })
}
</script>

<template>
  <ErrorPage
    code="401"
    :title="t('error.unauthorized')"
    :description="t('error.unauthorizedDesc')"
    emoji="馃攼"
  >
    <template #hint>
      {{ t('error.sessionExpiredHint') }}
    </template>
    <div class="mt-2">
      <Button size="lg" @click="goToLogin">
        <LogIn class="mr-2 h-4 w-4" />
        {{ t('error.loginNow') }}
      </Button>
    </div>
  </ErrorPage>
</template>
