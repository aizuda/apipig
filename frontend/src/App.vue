<script setup lang="ts">
import { RouterView } from 'vue-router'
import { Toast } from '@tabtab/ui'
</script>

<template>
  <Toast position="top-center" />
  <RouterView />
</template>
