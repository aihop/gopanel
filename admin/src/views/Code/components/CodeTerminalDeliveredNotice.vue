<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { NButton } from "naive-ui"
import { codeTerminalMessages } from "../codeTerminalMessages"

const { t } = useI18n({ messages: codeTerminalMessages })

defineEmits<{ newSession: [] }>()
</script>

<template>
  <!-- 会话已交付：worktree 连同 PTY 都已回收，这里没有终端可连，只给出下一步。 -->
  <div class="flex min-h-0 flex-1 items-center justify-center p-6 text-center text-slate-200">
    <div class="max-w-md">
      <div class="text-base font-medium">
        {{ t("code.deliveredSessionTerminalClosed") }}
      </div>
      <div class="mt-2 text-sm text-slate-400">
        {{ t("code.deliveredSessionTerminalHint") }}
      </div>
      <NButton
        class="mt-5"
        type="primary"
        @click="$emit('newSession')"
      >
        {{ t("code.createNextSession") }}
      </NButton>
    </div>
  </div>
</template>
