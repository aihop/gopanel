<template>
	<Teleport to="body">
		<div v-if="panels.length" class="quick-panel-layer">
			<section
				v-for="panel in panels"
				:key="panel.project.id"
				class="quick-panel"
				:style="panelStyle(panel)"
				@pointerdown="focusPanel(panel)"
			>
				<header class="quick-panel__header" @pointerdown="startDrag($event, panel)">
					<div class="flex min-w-0 items-center gap-2">
						<Icon name="mdi:dock-window" :size="18" />
						<span class="truncate font-semibold">{{ panel.project.name }}</span>
						<span class="quick-panel__label">{{ t("code.quickPanel") }}</span>
					</div>
					<div class="flex shrink-0 items-center gap-1" @pointerdown.stop>
						<n-tooltip>
							<template #trigger>
								<n-button quaternary circle size="small" @click="openFullProject(panel)">
									<template #icon><Icon name="mdi:open-in-new" :size="17" /></template>
								</n-button>
							</template>
							{{ t("code.openFullProject") }}
						</n-tooltip>
						<n-tooltip>
							<template #trigger>
								<n-button quaternary circle size="small" @click="closePanel(panel)">
									<template #icon><Icon name="mdi:close" :size="18" /></template>
								</n-button>
							</template>
							{{ t("code.closeQuickPanel") }}
						</n-tooltip>
					</div>
				</header>

				<div class="quick-panel__content">
					<Workspace
						:ref="instance => setWorkspaceRef(panel.project.id, instance)"
						:project-id="panel.project.id"
						embedded
						@close="closePanel(panel)"
					/>
				</div>

				<div
					v-for="direction in resizeDirections"
					:key="direction"
					class="quick-panel__resize-handle"
					:class="`quick-panel__resize-handle--${direction}`"
					@pointerdown.stop.prevent="startResize($event, panel, direction)"
				></div>
			</section>
		</div>
	</Teleport>
</template>

<script setup lang="ts">
import type { AIProject } from "@/api/interface/code"
import Icon from "@/components/common/Icon.vue"
import { codeProjectMessages } from "@/i18n/locales/codeProject"
import { onBeforeUnmount, onMounted, ref } from "vue"
import { onBeforeRouteLeave, useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import Workspace from "../Workspace.vue"

type ResizeDirection = "n" | "ne" | "e" | "se" | "s" | "sw" | "w" | "nw"
type WorkspaceInstance = { confirmClose: () => boolean }
type QuickPanel = {
	project: AIProject
	x: number
	y: number
	width: number
	height: number
	zIndex: number
}

const SCREEN_GAP = 12
const DEFAULT_WIDTH = 1040
const DEFAULT_HEIGHT = 720
const MIN_WIDTH = 640
const MIN_HEIGHT = 420
const resizeDirections: ResizeDirection[] = ["n", "ne", "e", "se", "s", "sw", "w", "nw"]

const router = useRouter()
const { t } = useI18n({ messages: codeProjectMessages })
const panels = ref<QuickPanel[]>([])
const workspaceRefs = new Map<number, WorkspaceInstance>()
let topZIndex = 1900
let activeCleanup: (() => void) | undefined

const viewportWidth = () => Math.max(1, window.innerWidth - SCREEN_GAP * 2)
const viewportHeight = () => Math.max(1, window.innerHeight - SCREEN_GAP * 2)
const minimumWidth = () => Math.min(MIN_WIDTH, viewportWidth())
const minimumHeight = () => Math.min(MIN_HEIGHT, viewportHeight())

function setWorkspaceRef(projectId: number, instance: unknown) {
	if (instance) workspaceRefs.set(projectId, instance as WorkspaceInstance)
	else workspaceRefs.delete(projectId)
}

function panelStyle(panel: QuickPanel) {
	return {
		left: `${panel.x}px`,
		top: `${panel.y}px`,
		width: `${panel.width}px`,
		height: `${panel.height}px`,
		zIndex: panel.zIndex
	}
}

function focusPanel(panel: QuickPanel) {
	panel.zIndex = ++topZIndex
}

function clampPanel(panel: QuickPanel) {
	panel.width = Math.min(Math.max(minimumWidth(), panel.width), viewportWidth())
	panel.height = Math.min(Math.max(minimumHeight(), panel.height), viewportHeight())
	panel.x = Math.min(Math.max(SCREEN_GAP, panel.x), Math.max(SCREEN_GAP, window.innerWidth - panel.width - SCREEN_GAP))
	panel.y = Math.min(Math.max(SCREEN_GAP, panel.y), Math.max(SCREEN_GAP, window.innerHeight - panel.height - SCREEN_GAP))
}

function open(project: AIProject) {
	const existing = panels.value.find(panel => panel.project.id === project.id)
	if (existing) {
		existing.project = project
		focusPanel(existing)
		return
	}
	const offset = panels.value.length % 6 * 28
	const width = Math.min(DEFAULT_WIDTH, viewportWidth())
	const height = Math.min(DEFAULT_HEIGHT, viewportHeight())
	const panel: QuickPanel = {
		project,
		x: Math.max(SCREEN_GAP, (window.innerWidth - width) / 2 + offset),
		y: Math.max(SCREEN_GAP, (window.innerHeight - height) / 2 + offset),
		width,
		height,
		zIndex: ++topZIndex
	}
	clampPanel(panel)
	panels.value.push(panel)
}

function canClose(panel: QuickPanel) {
	return workspaceRefs.get(panel.project.id)?.confirmClose() ?? true
}

function closePanel(panel: QuickPanel) {
	if (!canClose(panel)) return
	workspaceRefs.delete(panel.project.id)
	panels.value = panels.value.filter(item => item !== panel)
}

function openFullProject(panel: QuickPanel) {
	void router.push(`/code/project/${panel.project.id}`)
}

function bindPointerMove(move: (event: PointerEvent) => void) {
	activeCleanup?.()
	const stop = () => {
		window.removeEventListener("pointermove", move)
		window.removeEventListener("pointerup", stop)
		window.removeEventListener("pointercancel", stop)
		activeCleanup = undefined
	}
	window.addEventListener("pointermove", move)
	window.addEventListener("pointerup", stop)
	window.addEventListener("pointercancel", stop)
	activeCleanup = stop
}

function startDrag(event: PointerEvent, panel: QuickPanel) {
	if (event.button !== 0) return
	focusPanel(panel)
	const offsetX = event.clientX - panel.x
	const offsetY = event.clientY - panel.y
	bindPointerMove(moveEvent => {
		panel.x = moveEvent.clientX - offsetX
		panel.y = moveEvent.clientY - offsetY
		clampPanel(panel)
	})
}

function startResize(event: PointerEvent, panel: QuickPanel, direction: ResizeDirection) {
	if (event.button !== 0) return
	focusPanel(panel)
	const origin = { x: event.clientX, y: event.clientY, left: panel.x, top: panel.y, width: panel.width, height: panel.height }
	bindPointerMove(moveEvent => {
		const deltaX = moveEvent.clientX - origin.x
		const deltaY = moveEvent.clientY - origin.y
		let left = origin.left
		let top = origin.top
		let width = origin.width
		let height = origin.height
		if (direction.includes("e")) width = origin.width + deltaX
		if (direction.includes("s")) height = origin.height + deltaY
		if (direction.includes("w")) {
			width = origin.width - deltaX
			left = origin.left + deltaX
		}
		if (direction.includes("n")) {
			height = origin.height - deltaY
			top = origin.top + deltaY
		}
		const right = origin.left + origin.width
		const bottom = origin.top + origin.height
		panel.width = Math.min(Math.max(minimumWidth(), width), viewportWidth())
		panel.height = Math.min(Math.max(minimumHeight(), height), viewportHeight())
		panel.x = direction.includes("w") ? right - panel.width : left
		panel.y = direction.includes("n") ? bottom - panel.height : top
		clampPanel(panel)
	})
}

function handleViewportResize() {
	panels.value.forEach(clampPanel)
}

onBeforeRouteLeave(() => panels.value.every(canClose))
onMounted(() => window.addEventListener("resize", handleViewportResize))
onBeforeUnmount(() => {
	activeCleanup?.()
	window.removeEventListener("resize", handleViewportResize)
})

defineExpose({ open })
</script>

<style scoped>
.quick-panel-layer { position: fixed; inset: 0; z-index: 1899; pointer-events: none; }
.quick-panel { position: absolute; display: flex; flex-direction: column; overflow: hidden; pointer-events: auto; border: 1px solid rgb(59 130 246 / 35%); border-radius: 18px; background: var(--n-color, #fff); box-shadow: 0 24px 70px rgb(15 23 42 / 28%); }
.quick-panel__header { display: flex; min-height: 46px; align-items: center; justify-content: space-between; gap: 12px; padding: 7px 8px 7px 14px; cursor: move; touch-action: none; user-select: none; border-bottom: 1px solid rgb(59 130 246 / 18%); background: color-mix(in srgb, var(--n-color, white) 88%, #dbeafe 12%); color: var(--n-text-color); }
.quick-panel__label { border-radius: 9999px; background: rgb(59 130 246 / 10%); padding: 2px 7px; font-size: 10px; color: #2563eb; }
.quick-panel__content { min-height: 0; flex: 1; overflow: hidden; }
.quick-panel__resize-handle { position: absolute; z-index: 2; touch-action: none; }
.quick-panel__resize-handle--n, .quick-panel__resize-handle--s { right: 10px; left: 10px; height: 8px; cursor: ns-resize; }
.quick-panel__resize-handle--n { top: -4px; } .quick-panel__resize-handle--s { bottom: -4px; }
.quick-panel__resize-handle--e, .quick-panel__resize-handle--w { top: 10px; bottom: 10px; width: 8px; cursor: ew-resize; }
.quick-panel__resize-handle--e { right: -4px; } .quick-panel__resize-handle--w { left: -4px; }
.quick-panel__resize-handle--ne, .quick-panel__resize-handle--se, .quick-panel__resize-handle--sw, .quick-panel__resize-handle--nw { width: 14px; height: 14px; }
.quick-panel__resize-handle--ne { top: -5px; right: -5px; cursor: nesw-resize; } .quick-panel__resize-handle--se { right: -5px; bottom: -5px; cursor: nwse-resize; }
.quick-panel__resize-handle--sw { bottom: -5px; left: -5px; cursor: nesw-resize; } .quick-panel__resize-handle--nw { top: -5px; left: -5px; cursor: nwse-resize; }
</style>
