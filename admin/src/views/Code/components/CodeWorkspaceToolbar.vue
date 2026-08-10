<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import CodeApprovalCenter from "./CodeApprovalCenter.vue"
import CodeDeliveryPanel from "./CodeDeliveryPanel.vue"
import CodeTaskDeliveryButton from "./CodeTaskDeliveryButton.vue"
import SessionApprovalPolicy from "./SessionApprovalPolicy.vue"
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
}>()

const emit = defineEmits<{
	showStructure: []
	takeTerminal: []
	openHistory: []
	toggleFullscreen: []
	openFile: [file: { path: string; extension: string }]
	updateMode: [mode: CodeWorkspaceMode]
	goHome: []
}>()

const { t } = useI18n({ messages: codeWorkspaceMessages })
const sessionIcon = computed(() => (props.isTerminalSession ? "mdi:console-line" : "mdi:robot-outline"))
</script>

<template>
	<div
		class="mb-2 flex shrink-0 flex-col gap-2 rounded-2xl border border-slate-200/80 bg-white/90 p-2.5 shadow-sm backdrop-blur md:flex-row md:items-center md:justify-between dark:border-[var(--border-color)] dark:bg-[var(--bg-default-color)]"
	>
		<div class="flex min-w-0 items-center gap-3 px-1">
			<div
				class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-slate-100 text-slate-600 dark:bg-white/10 dark:text-slate-300"
			>
				<Icon :name="sessionIcon" :size="19" />
			</div>
			<div class="min-w-0">
				<div class="flex items-center gap-2">
					<div class="truncate text-sm font-semibold text-[var(--n-text-color)]">{{ sessionLabel }}</div>
					<n-tag
						v-if="hasContext"
						size="tiny"
						:bordered="false"
						:type="isTerminalSession ? 'default' : 'info'"
					>
						{{ t(isTerminalSession ? "code.terminalSession" : "code.aiSession") }}
					</n-tag>
				</div>
				<div class="truncate text-xs text-[var(--n-text-color-3)]">{{ sessionSubtitle }}</div>
			</div>
		</div>

		<div class="flex min-w-0 flex-wrap items-center gap-2 md:justify-end">
			<CodeTaskDeliveryButton v-if="sessionId !== null && !isTerminalSession" :session-id="sessionId" />
			<div
				v-if="hasContext && !isTerminalSession"
				class="rounded-xl bg-slate-50/80 dark:bg-white/5"
			>
				<WorkspaceModeSwitch :value="workspaceMode" @update:value="emit('updateMode', $event)" />
			</div>

			<CodeApprovalCenter
				v-if="hasContext && !isTerminalSession"
				:session-id="sessionId"
				@take-terminal="emit('takeTerminal')"
			/>

			<n-popover
				v-if="sessionId !== null && !isTerminalSession"
				trigger="click"
				placement="bottom-end"
				style="width: 300px"
			>
				<template #trigger>
					<n-button size="small" secondary class="!rounded-xl">
						<template #icon><Icon name="mdi:tune-variant" /></template>
						{{ t("code.aiTools") }}
					</n-button>
				</template>
				<div class="space-y-4 p-1">
					<div>
						<div class="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--n-text-color-3)]">
							{{ t("code.deliveryAndQuality") }}
						</div>
						<CodeDeliveryPanel :session-id="sessionId" @open-file="emit('openFile', $event)" />
					</div>
					<div class="border-t border-slate-200 pt-4 dark:border-[var(--border-color)]">
						<SessionApprovalPolicy :session-id="sessionId" />
					</div>
				</div>
			</n-popover>

			<div
				class="flex items-center rounded-xl border border-slate-200 bg-white p-0.5 dark:border-[var(--border-color)] dark:bg-white/5"
			>
				<n-tooltip v-if="sessionId !== null && workspaceMode === 'editor'">
					<template #trigger>
						<n-button quaternary circle size="small" class="xl:hidden" @click="emit('showStructure')">
							<template #icon><Icon name="mdi:file-tree-outline" /></template>
						</n-button>
					</template>
					{{ t("code.projectStructure") }}
				</n-tooltip>
				<n-tooltip v-if="hasContext && !isTerminalSession">
					<template #trigger>
						<n-button quaternary circle size="small" @click="emit('openHistory')">
							<template #icon><Icon name="mdi:message-text-clock-outline" /></template>
						</n-button>
					</template>
					{{ t("code.conversationHistory") }}
				</n-tooltip>
				<n-tooltip v-if="hasContext">
					<template #trigger>
						<n-button quaternary circle size="small" @click="emit('goHome')">
							<template #icon><Icon name="mdi:view-dashboard-outline" /></template>
						</n-button>
					</template>
					{{ t("code.taskHome") }}
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
