<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '@tabtab/utils'
import { PanelLeftClose, PanelRight } from '@lucide/vue'
import { Button } from '../../ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../../ui/tooltip'
import { useLayout } from '../utils'

interface Props {
  class?: HTMLAttributes['class']
  showTooltip?: boolean
  labelShow?: string
  labelHide?: string
}

const props = withDefaults(defineProps<Props>(), {
  showTooltip: true,
  labelShow: '展开侧栏',
  labelHide: '收起侧栏',
})

const { hidden, toggleHidden } = useLayout()
</script>

<template>
  <TooltipProvider v-if="props.showTooltip" :delay-duration="200">
    <Tooltip>
      <TooltipTrigger as-child>
        <Button
          variant="ghost"
          size="icon"
          :class="cn('h-8 w-8', props.class)"
          @click="toggleHidden"
        >
          <PanelRight v-if="hidden" class="h-4 w-4" />
          <PanelLeftClose v-else class="h-4 w-4" />
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom">
        <span>{{ hidden ? props.labelShow : props.labelHide }}</span>
      </TooltipContent>
    </Tooltip>
  </TooltipProvider>

  <Button
    v-else
    variant="ghost"
    size="icon"
    :class="cn('h-8 w-8', props.class)"
    @click="toggleHidden"
  >
    <PanelRight v-if="hidden" class="h-4 w-4" />
    <PanelLeftClose v-else class="h-4 w-4" />
  </Button>
</template>
