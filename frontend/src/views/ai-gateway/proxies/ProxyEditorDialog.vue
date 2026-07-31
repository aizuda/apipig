<script setup lang="ts">
import { computed, toRef } from 'vue'
import { Button, Dialog, DialogFixedContent, Input, Label } from '@tabtab/ui'
import type { ProxyNode } from '@/api/ai-gateway'

const props = defineProps<{
  open: boolean
  loading: boolean
  errorMessage: string
  form: ProxyNode
}>()

const emit = defineEmits<{
  close: []
  submit: []
}>()

const proxyForm = toRef(props, 'form')
const editorOpen = computed(() => props.open)
const editorErrorMessage = computed(() => props.errorMessage)

function closeEditor() {
  emit('close')
}

function saveProxy() {
  emit('submit')
}
</script>

<template>
  <Dialog
    :open="editorOpen"
    @update:open="
      (open) => {
        if (!open) closeEditor()
      }
    "
  >
    <DialogFixedContent
      :title="proxyForm.id ? '编辑 IP 代理' : '新增 IP 代理'"
      description="配置代理协议、网络地址和可选认证信息。"
      class="sm:max-w-xl"
    >
      <div class="space-y-3">
        <div
          v-if="editorErrorMessage"
          class="sticky top-0 z-20 rounded-md border border-destructive/40 bg-background px-4 py-3 text-sm text-destructive shadow-sm"
        >
          {{ editorErrorMessage }}
        </div>
        <div class="space-y-1.5">
          <Label for="proxy-name">代理名称 <span class="text-destructive">*</span></Label
          ><Input id="proxy-name" v-model="proxyForm.name" placeholder="例如 香港出口代理" />
          <p class="text-xs text-muted-foreground">用于渠道配置时识别该代理线路。</p>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="proxy-scheme">代理协议 <span class="text-destructive">*</span></Label
            ><select
              id="proxy-scheme"
              v-model="proxyForm.scheme"
              class="h-10 w-full rounded-md border bg-background px-3 text-sm"
            >
              <option value="http">HTTP</option>
              <option value="https">HTTPS</option>
              <option value="socks5">SOCKS5</option>
            </select>
            <p class="text-xs text-muted-foreground">必须与代理服务实际监听协议一致。</p>
          </div>
          <div class="space-y-1.5">
            <Label for="proxy-region">区域标识</Label
            ><Input id="proxy-region" v-model="proxyForm.region" placeholder="例如 中国香港" />
            <p class="text-xs text-muted-foreground">仅用于管理和筛选，不影响网络连接。</p>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-[1fr_140px]">
          <div class="space-y-1.5">
            <Label for="proxy-host">代理主机 <span class="text-destructive">*</span></Label
            ><Input id="proxy-host" v-model="proxyForm.host" placeholder="域名或 IP 地址" />
            <p class="text-xs text-muted-foreground">不要填写协议前缀或路径。</p>
          </div>
          <div class="space-y-1.5">
            <Label for="proxy-port">端口 <span class="text-destructive">*</span></Label
            ><Input
              id="proxy-port"
              v-model.number="proxyForm.port"
              type="number"
              min="1"
              max="65535"
            />
            <p class="text-xs text-muted-foreground">范围 1–65535。</p>
          </div>
        </div>
        <div class="grid gap-3 sm:grid-cols-2">
          <div class="space-y-1.5">
            <Label for="proxy-username">认证用户名</Label
            ><Input id="proxy-username" v-model="proxyForm.username" placeholder="无认证可留空" />
            <p class="text-xs text-muted-foreground">代理无需身份认证时留空。</p>
          </div>
          <div class="space-y-1.5">
            <Label for="proxy-password">认证密码</Label
            ><Input
              id="proxy-password"
              v-model="proxyForm.password"
              type="password"
              autocomplete="new-password"
              placeholder="无认证可留空"
            />
            <p class="text-xs text-muted-foreground">
              密码会加密保存；编辑时保留脱敏值表示不修改。
            </p>
          </div>
        </div>
        <div class="space-y-1.5">
          <Label for="proxy-status">启用状态</Label
          ><select
            id="proxy-status"
            v-model.number="proxyForm.status"
            class="h-10 w-full rounded-md border bg-background px-3 text-sm"
          >
            <option :value="1">启用代理</option>
            <option :value="2">禁用代理</option>
          </select>
          <p class="text-xs text-muted-foreground">禁用后已关联渠道将不再使用该代理。</p>
        </div>
      </div>
      <template #footer>
        <Button variant="outline" :disabled="loading" @click="closeEditor">取消</Button>
        <Button :disabled="loading" @click="saveProxy">{{
          proxyForm.id ? '保存修改' : '创建代理'
        }}</Button>
      </template>
    </DialogFixedContent>
  </Dialog>
</template>
