<script setup lang="ts">
import type { ListboxFilterProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { Search } from '@lucide/vue'
import { ListboxFilter, useForwardProps } from 'reka-ui'
import { cn } from '@tabtab/utils'
import { useCommand } from '.'
import { onMounted, watch } from 'vue'

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<
  ListboxFilterProps & {
    class?: HTMLAttributes['class']
    defaultValue?: string
  }
>()

const delegatedProps = reactiveOmit(props, 'class', 'defaultValue')

const forwardedProps = useForwardProps(delegatedProps)

const { filterState } = useCommand()

onMounted(() => {
  if (props.defaultValue !== undefined && props.defaultValue !== '') {
    filterState.search = props.defaultValue
  }
})

watch(
  () => props.defaultValue,
  (value) => {
    if (value !== undefined && value !== '') {
      filterState.search = value
    }
  },
)
</script>

<template>
  <div
    data-slot="command-input-wrapper"
    class="flex h-12 items-center gap-3 border-b border-border/50 px-4"
  >
    <Search class="h-5 w-5 shrink-0 text-muted-foreground" />
    <ListboxFilter
      v-bind="{ ...forwardedProps, ...$attrs }"
      v-model="filterState.search"
      data-slot="command-input"
      auto-focus
      :class="
        cn(
          'placeholder:text-muted-foreground flex h-full w-full bg-transparent text-sm outline-none disabled:cursor-not-allowed disabled:opacity-50',
          props.class,
        )
      "
    />
  </div>
</template>
