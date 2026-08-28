<script setup lang="ts">
import { computed } from 'vue'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  Button,
} from '@tabtab/ui'

const props = withDefaults(
  defineProps<{
    deleteTarget?: { name: string } | null
    statusTarget?: { item: { name: string }; targetStatus: number } | null
    resetTokenTarget?: { name: string } | null
    loading?: boolean
    statusChanging?: boolean
    tokenResetting?: boolean
    deleteDescription?: string
  }>(),
  {
    deleteTarget: null,
    statusTarget: null,
    resetTokenTarget: null,
    loading: false,
    statusChanging: false,
    tokenResetting: false,
    deleteDescription: '删除后无法恢复；存在关联配置时后端会阻止删除并返回原因。',
  },
)

const emit = defineEmits<{
  closeDelete: []
  confirmDelete: []
  closeStatus: []
  confirmStatus: []
  closeTokenReset: []
  confirmTokenReset: []
}>()

const statusBusy = computed(() => props.loading || props.statusChanging)

function statusLabel(status?: number) {
  return status === 1 ? '启用' : '禁用'
}
</script>

<template>
  <AlertDialog
    :open="Boolean(deleteTarget)"
    @update:open="(open) => !open && !loading && emit('closeDelete')"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>确认删除 {{ deleteTarget?.name }}？</AlertDialogTitle>
        <AlertDialogDescription>{{ deleteDescription }}</AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="loading" @click="emit('closeDelete')">取消</AlertDialogCancel>
        <Button
          type="button"
          variant="destructive"
          :disabled="loading"
          @click="emit('confirmDelete')"
        >
          确认删除
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <AlertDialog
    :open="Boolean(statusTarget)"
    @update:open="(open) => !open && !statusBusy && emit('closeStatus')"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>确认切换 {{ statusTarget?.item.name }} 状态？</AlertDialogTitle>
        <AlertDialogDescription>
          是否将“{{ statusTarget?.item.name }}”切换为“{{
            statusLabel(statusTarget?.targetStatus)
          }}”状态？确认后将立即调用后台接口。
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="statusBusy" @click="emit('closeStatus')"
          >取消</AlertDialogCancel
        >
        <Button type="button" :disabled="statusBusy" @click="emit('confirmStatus')">
          确认{{ statusLabel(statusTarget?.targetStatus) }}
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>

  <AlertDialog
    :open="Boolean(resetTokenTarget)"
    @update:open="(open) => !open && !tokenResetting && emit('closeTokenReset')"
  >
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>确认重置 {{ resetTokenTarget?.name }} 的 API 密钥？</AlertDialogTitle>
        <AlertDialogDescription>
          重置后旧密钥将立即失效，使用旧密钥的服务会停止工作。新密钥只展示一次，请及时更新相关服务配置。
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel :disabled="tokenResetting" @click="emit('closeTokenReset')">
          取消
        </AlertDialogCancel>
        <Button
          type="button"
          variant="destructive"
          :disabled="tokenResetting"
          @click="emit('confirmTokenReset')"
        >
          {{ tokenResetting ? '重置中...' : '确认重置' }}
        </Button>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
