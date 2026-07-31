<script setup lang="ts">
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '../../ui/dropdown-menu'
import { Button } from '../../ui/button'
import { Avatar, AvatarFallback, AvatarImage } from '../../ui/avatar'
import { ChevronUp, Settings, LogOut } from '@lucide/vue'
import { computed } from 'vue'

interface Props {
  /** User name */
  userName?: string
  /** User email */
  userEmail?: string
  /** Avatar image URL */
  avatarSrc?: string
  /** Avatar fallback text */
  avatarFallback?: string
  /** Is sidebar collapsed */
  collapsed?: boolean
  /** Label for logged in status (translated) */
  labelLoggedIn?: string
  /** Label for account settings menu item (translated) */
  labelAccountSettings?: string
  /** Label for logout menu item (translated) */
  labelLogout?: string
}

const props = withDefaults(defineProps<Props>(), {
  userName: 'Admin',
  userEmail: '',
  avatarSrc: '',
  avatarFallback: '',
  collapsed: false,
  labelLoggedIn: 'Logged in',
  labelAccountSettings: 'Account Settings',
  labelLogout: 'Log out',
})

const emit = defineEmits<{
  /** Trigger logout */
  (e: 'logout'): void
  /** Trigger settings */
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
  <!-- 鎶樺彔鐘舵€侊細鐐瑰嚮澶村儚鏄剧ず涓嬫媺鑿滃崟 -->
  <div
    v-if="collapsed"
    class="py-2 px-1 flex flex-col items-center border-t border-border/30 overflow-hidden"
  >
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <button class="group relative p-1 rounded-lg hover:bg-accent transition-all duration-200">
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

  <!-- Expanded state -->
  <div v-else class="px-2 py-2 border-t border-border/30 overflow-hidden">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button
          variant="ghost"
          class="w-full justify-start gap-2 h-auto py-2 px-2 transition-all duration-200 hover:bg-primary/5 group"
          aria-label="User menu"
        >
          <Avatar class="h-7 w-7">
            <AvatarImage :src="avatarSrc" :alt="userName" />
            <AvatarFallback
              class="text-xs font-medium bg-gradient-to-br from-primary/80 to-primary text-primary-foreground"
            >
              {{ initials }}
            </AvatarFallback>
          </Avatar>
          <div class="flex flex-col min-w-0 flex-1 text-left overflow-hidden">
            <span
              class="text-sm font-medium truncate group-hover:text-primary transition-colors duration-200"
            >
              {{ userName }}
            </span>
            <span class="text-[10px] text-muted-foreground truncate">
              {{ userEmail }}
            </span>
          </div>
          <ChevronUp
            class="h-4 w-4 text-muted-foreground flex-shrink-0 transition-transform duration-200 group-data-[state=open]:rotate-180"
          />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" class="w-56 shadow-xl border-border/50" :side-offset="8">
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
