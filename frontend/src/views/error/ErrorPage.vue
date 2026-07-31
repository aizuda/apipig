<script setup lang="ts">
import type { Component } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ArrowLeft, Home } from '@lucide/vue'
import { Button, Card, CardContent, CardDescription, CardHeader, CardTitle } from '@tabtab/ui'

const { t } = useI18n()

interface ErrorPageProps {
  code: string
  title: string
  description: string
  emoji?: string
  icon?: Component
}

const props = defineProps<ErrorPageProps>()

const router = useRouter()

function goHome() {
  router.push({ name: 'Dashboard' })
}

function goBack() {
  router.back()
}
</script>

<template>
  <div
    class="flex min-h-screen items-center justify-center bg-gradient-to-br from-primary/5 via-background to-secondary/5 p-4"
  >
    <Card class="w-full max-w-md border-border/50 bg-muted/30 text-center">
      <CardHeader>
        <div v-if="props.emoji" class="mb-4 text-6xl">
          {{ props.emoji }}
        </div>
        <div v-else-if="props.icon" class="mb-4 flex justify-center">
          <component :is="props.icon" class="h-16 w-16 text-muted-foreground" />
        </div>
        <CardTitle class="text-2xl"> {{ props.code }} - {{ props.title }} </CardTitle>
        <CardDescription>{{ props.description }}</CardDescription>
      </CardHeader>
      <CardContent class="space-y-4">
        <p class="text-muted-foreground">
          <slot name="hint">
            {{ t('error.checkUrlHint') }}
          </slot>
        </p>
        <div class="flex justify-center gap-2">
          <Button variant="outline" @click="goBack">
            <ArrowLeft class="mr-2 h-4 w-4" />
            {{ t('error.back') }}
          </Button>
          <Button @click="goHome">
            <Home class="mr-2 h-4 w-4" />
            {{ t('error.home') }}
          </Button>
        </div>
        <slot />
      </CardContent>
    </Card>
  </div>
</template>
