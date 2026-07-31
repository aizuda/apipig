<script setup lang="ts">
import type { DialogContentEmits, DialogContentProps } from 'reka-ui'
import type { HTMLAttributes } from 'vue'
import { reactiveOmit } from '@vueuse/core'
import { cn } from '@tabtab/utils'
import { useForwardPropsEmits } from 'reka-ui'
import DialogContent from './DialogContent.vue'
import DialogDescription from './DialogDescription.vue'
import DialogFooter from './DialogFooter.vue'
import DialogHeader from './DialogHeader.vue'
import DialogTitle from './DialogTitle.vue'

defineOptions({
  inheritAttrs: false,
})

const props = defineProps<
  DialogContentProps & {
    bodyClass?: HTMLAttributes['class']
    class?: HTMLAttributes['class']
    description?: string
    title: string
  }
>()
const emits = defineEmits<DialogContentEmits>()

const delegatedProps = reactiveOmit(props, 'bodyClass', 'class', 'description', 'title')
const forwarded = useForwardPropsEmits(delegatedProps, emits)
</script>

<template>
  <DialogContent
    v-bind="{ ...$attrs, ...forwarded }"
    :class="
      cn(
        'grid max-h-[calc(100dvh-2rem)] grid-rows-[auto_minmax(0,1fr)_auto] gap-0 overflow-hidden p-0',
        props.class,
      )
    "
  >
    <div class="shrink-0 border-b px-6 py-5 pr-12">
      <DialogHeader class="gap-1.5 text-left">
        <DialogTitle>{{ props.title }}</DialogTitle>
        <DialogDescription v-if="props.description">
          {{ props.description }}
        </DialogDescription>
      </DialogHeader>
      <slot name="header" />
    </div>

    <div :class="cn('min-h-0 overflow-y-auto px-6 py-4', props.bodyClass)">
      <slot />
    </div>

    <div v-if="$slots.footer" class="shrink-0 border-t bg-muted/20 px-6 py-4">
      <DialogFooter>
        <slot name="footer" />
      </DialogFooter>
    </div>
  </DialogContent>
</template>
