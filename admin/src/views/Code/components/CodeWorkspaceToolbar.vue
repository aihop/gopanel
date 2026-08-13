<script setup lang="ts">
import { computed, nextTick, ref } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import CodeApprovalCenter from "./CodeApprovalCenter.vue"
import CodeDeliveryPanel from "./CodeDeliveryPanel.vue"
import CodeTaskDeliveryButton from "./CodeTaskDeliveryButton.vue"
import SessionApprovalPolicy from "./SessionApprovalPolicy.vue"
import CodeMemoryEntryButton from "./CodeMemoryEntryButton.vue"
import CodeMemoryManager from "./CodeMemoryManager.vue"
import WorkspaceModeSwitch, { type CodeWorkspaceMode } from "./WorkspaceModeSwitch.vue"
import { codeWorkspaceMessages } from "../codeWorkspaceMessages"

const props = defineProps<{
	sessionLabel: string
	sessionSubtitle: string
	sessionId: number | null
	hasContext: boolean
	isTerminalSession: boolean
	workspaceMode: CodeWorkspaceMode
	embedded: boolean
	fullscreenLabel: string
	isFullscreen: boolean
	projectId: number
}>()

const emit = defineEmits<{
	showStructure: []
	takeTerminal: []
	openHistory: []
	toggleFullscreen: []
	openFile: [file: { path: string; extension: string }]
	updateMode: [mode: CodeWorkspaceMode]
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const sessionIcon = computed(() => (props.isTerminalSession ? "mdi:console-line" : "mdi:robot-outline"))
const showAITools = ref(false)
const memoryManager = ref<InstanceType<typeof CodeMemoryManager> | null>(null)

async function openMemory() {
	showAITools.value = false
	await nextTick()
	memoryManager.value?.open()
}
</script>

<template>
  <div
    class="mb-2 flex shrink-0 flex-col gap-2 p-2.5 backdrop-blur md:flex-row md:items-center md:justify-between dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
  >
    <div class="flex min-w-0 items-center gap-3 px-1">
      <div
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600 dark:bg-white/10 dark:text-slate-300"
      >
        <Icon
          :name="sessionIcon"
          :size="19"
        />
      </div>
      <div class="min-w-0">
        <div class="flex items-center gap-2">
          <div class="truncate text-sm font-semibold text-[var(--n-text-color)]">
            {{ sessionLabel }}
          </div>
          <n-tag
            v-if="hasContext"
            size="tiny"
            :bordered="false"
            :type="isTerminalSession ? 'default' : 'info'"
          >
            {{ t(isTerminalSession ? "code.terminalSession" : "code.aiSession") }}
          </n-tag>
        </div>
        <div class="truncate text-xs text-gray-500">
          {{ sessionSubtitle }}
        </div>
      </div>
    </div>

    <div class="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">
      <CodeTaskDeliveryButton
        v-if="sessionId !== null && !isTerminalSession"
        :session-id="sessionId"
      />
      <div
        v-if="hasContext && !isTerminalSession"
        class="rounded-xl bg-slate-50/80 dark:bg-white/5"
      >
        <WorkspaceModeSwitch
          :value="workspaceMode"
          @update:value="emit('updateMode', $event)"
        />
      </div>

      <CodeApprovalCenter
        v-if="hasContext && !isTerminalSession"
        :session-id="sessionId"
        @take-terminal="emit('takeTerminal')"
      />

      <n-popover
        v-if="sessionId !== null && !isTerminalSession"
        :show="showAITools"
        trigger="click"
        placement="bottom-end"
        style="width: 300px"
        @update:show="showAITools = $event"
      >
        <template #trigger>
          <n-button
            size="small"
            secondary
            class="!rounded-xl"
          >
            <template #icon>
              <Icon name="mdi:tune-variant" />
            </template>
            {{ t("code.aiTools") }}
          </n-button>
        </template>
        <div class="space-y-4 p-1">
          <div>
            <div class="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--n-text-color-3)]">
              {{ t("code.deliveryAndQuality") }}
            </div>
            <CodeDeliveryPanel
              :session-id="sessionId"
              @open-file="emit('openFile', $event)"
            />
          </div>
          <div class="border-t border-slate-200 pt-4 dark:border-[var(--border-color)]">
            <SessionApprovalPolicy :session-id="sessionId" />
          </div>
          <div class="border-t border-slate-200 pt-4 dark:border-[var(--border-color)]">
            <div class="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--n-text-color-3)]">
              {{ t("code.context") }}
            </div>
            <CodeMemoryEntryButton @open="openMemory" />
          </div>
        </div>
      </n-popover>

      <CodeMemoryManager
        ref="memoryManager"
        :project-id="projectId"
      />

      <div
        class="flex items-center rounded-xl border border-slate-200 bg-white p-0.5 dark:border-[var(--border-color)] dark:bg-white/5"
      >
        <n-tooltip v-if="hasContext && !isTerminalSession">
          <template #trigger>
            <n-button
              quaternary
              circle
              size="small"
              @click="emit('openHistory')"
            >
              <template #icon>
                <Icon name="mdi:message-text-clock-outline" />
              </template>
            </n-button>
          </template>
          {{ t("code.conversationHistory") }}
        </n-tooltip>
        <n-tooltip v-if="!embedded">
          <template #trigger>
            <n-button
              quaternary
              circle
              size="small"
              :aria-label="fullscreenLabel"
              @click="emit('toggleFullscreen')"
            >
              <template #icon>
                <Icon
                  :name="
                    isFullscreen
                      ? 'fluent:full-screen-minimize-24-regular'
                      : 'fluent:full-screen-maximize-24-regular'
                  "
                />
              </template>
            </n-button>
          </template>
          {{ fullscreenLabel }}
        </n-tooltip>
      </div>
    </div>
  </div>
</template>
