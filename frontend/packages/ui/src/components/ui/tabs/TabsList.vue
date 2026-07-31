<script setup lang="ts">
import type { TabsListProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { TabsList } from 'reka-ui'
import { cn } from '@tabtab/utils'

const props = defineProps<TabsListProps & { class?: HTMLAttributes['class'] }>()

const delegatedProps = reactiveOmit(props, 'class')
</script>

<template>
  <TabsList
    data-slot="tabs-list"
    v-bind="delegatedProps"
    :class="
      cn(
        'bg-muted/50 backdrop-blur-sm supports-[backdrop-filter]:bg-muted/30 text-muted-foreground inline-flex h-9 w-fit items-center justify-center rounded-lg p-[3px] border border-border/50',
        props.class,
      )
    "
  >
    <slot />
  </TabsList>
</template>
