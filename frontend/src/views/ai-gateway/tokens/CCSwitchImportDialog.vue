<script setup lang="ts">
import { ref, watch } from 'vue'
import { Import as ImportIcon } from '@lucide/vue'
import { Button, Dialog, DialogFixedContent, Input, Label } from '@tabtab/ui'
import {
  CC_SWITCH_APPS,
  createCCSwitchImportDefaults,
  type CCSwitchApp,
  type CCSwitchImportSelection,
} from './cc-switch'

const props = defineProps<{
  open: boolean
  tokenName: string
  models: string[]
}>()

const emit = defineEmits<{
  close: []
  confirm: [selection: CCSwitchImportSelection]
}>()

const selectedApp = ref<CCSwitchApp>('codex')
const importName = ref('')
const selectedModel = ref('')

watch(
  () => props.open,
  (open) => {
    if (!open) return
    const defaults = createCCSwitchImportDefaults(props.tokenName, props.models)
    selectedApp.value = defaults.app
    importName.value = defaults.name
    selectedModel.value = defaults.model || ''
  },
)

function closeDialog() {
  emit('close')
}

function confirmImport() {
  const name = importName.value.trim()
  if (!name) return
  emit('confirm', {
    app: selectedApp.value,
    name,
    model: selectedModel.value || undefined,
  })
}
</script>

<template>
  <Dialog
    :open="open"
    @update:open="
      (nextOpen) => {
        if (!nextOpen) closeDialog()
      }
    "
  >
    <DialogFixedContent
      title="导入到 CC Switch"
      description="确认供应商名称、目标应用和主模型。"
      class="sm:max-w-lg"
    >
      <div class="space-y-4">
        <div class="space-y-1.5">
          <Label for="cc-switch-import-name">名称</Label>
          <Input
            id="cc-switch-import-name"
            v-model="importName"
            maxlength="80"
            autocomplete="off"
          />
        </div>

        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="cc-switch-import-app">应用</Label>
            <select
              id="cc-switch-import-app"
              v-model="selectedApp"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option v-for="app in CC_SWITCH_APPS" :key="app.value" :value="app.value">
                {{ app.label }}
              </option>
            </select>
          </div>

          <div class="space-y-1.5">
            <Label for="cc-switch-import-model">主模型</Label>
            <select
              id="cc-switch-import-model"
              v-model="selectedModel"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option value="">不指定主模型</option>
              <option v-for="model in models" :key="model" :value="model">
                {{ model }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <template #footer>
        <Button variant="outline" @click="closeDialog">取消</Button>
        <Button :disabled="!importName.trim()" @click="confirmImport">
          <ImportIcon class="mr-1.5 h-4 w-4" />
          打开 CC Switch
        </Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
