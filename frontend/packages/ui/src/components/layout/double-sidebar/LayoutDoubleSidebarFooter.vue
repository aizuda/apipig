<script setup lang="ts">
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../../ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from '../../ui/avatar'
import { Settings, LogOut } from '@lucide/vue'
import { computed } from 'vue'

interface Props {
  userName?: string
  userEmail?: string
  avatarSrc?: string
  avatarFallback?: string
  labelLoggedIn?: string
  labelAccountSettings?: string
  labelLogout?: string
}

const props = withDefaults(defineProps<Props>(), {
  userName: 'Admin',
  userEmail: '',
  avatarSrc: '',
  avatarFallback: '',
  labelLoggedIn: 'Logged in',
  labelAccountSettings: 'Account Settings',
  labelLogout: 'Log out',
})

const emit = defineEmits<{
  (e: 'logout'): void
  (e: 'settings'): void
}>()

const initials = computed(() => {
  if (props.avatarFallback) return props.avatarFallback
  return props.userName?.charAt(0)?.toUpperCase() || 'A'
})

function handleLogout() {
  emit('logout')
}

function handleSettings() {
  emit('settings')
}
</script>

<template>
  <div class="border-t border-border/30 p-2">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <button class="group relative p-1.5 rounded-lg hover:bg-accent transition-all duration-200">
          <Avatar class="h-7 w-7">
            <AvatarImage :src="avatarSrc" :alt="userName" />
            <AvatarFallback
              class="text-xs font-medium bg-gradient-to-br from-primary/80 to-primary text-primary-foreground"
            >
              {{ initials }}
            </AvatarFallback>
          </Avatar>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="start"
        side="right"
        class="w-56 shadow-xl border-border/50"
        :side-offset="8"
      >
        <div class="px-3 py-2">
          <p class="text-xs font-medium text-muted-foreground">
            {{ labelLoggedIn }}
          </p>
          <p class="text-sm font-semibold truncate">
            {{ userEmail }}
          </p>
        </div>
        <DropdownMenuSeparator />
        <DropdownMenuItem class="gap-2 cursor-pointer" @click="handleSettings">
          <Settings class="h-4 w-4" />
          <span>{{ labelAccountSettings }}</span>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          class="gap-2 cursor-pointer text-destructive focus:text-destructive"
          @click="handleLogout"
        >
          <LogOut class="h-4 w-4" />
          <span>{{ labelLogout }}</span>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  </div>
</template>
