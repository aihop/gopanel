<template>
	<div class="tool-dock-layer">
		<div v-show="activeTool" class="tool-dock-panel" :style="panelStyle">
			<keep-alive>
				<SystemDiagnosticPanel v-if="activeTool === 'diagnostic'" @close="activeTool = null" />
				<FloatingNote v-else-if="activeTool === 'note'" @close="activeTool = null" />
			</keep-alive>
		</div>

		<nav class="tool-dock" :style="dockStyle">
			<n-tooltip placement="left">
				<template #trigger>
					<button class="tool-dock__handle" type="button" :aria-label="t('systemDiagnostic.drag')" @pointerdown="startDrag">
						<Icon name="mdi:drag-vertical" :size="18" />
					</button>
				</template>
				{{ t("systemDiagnostic.drag") }}
			</n-tooltip>
			<n-tooltip v-if="canDiagnose" placement="left">
				<template #trigger>
					<button class="tool-dock__button is-diagnostic" :class="{ 'is-active': activeTool === 'diagnostic' }" type="button" @click="toggle('diagnostic')">
						<Icon name="mdi:stethoscope" :size="22" />
					</button>
				</template>
				{{ t("systemDiagnostic.open") }}
			</n-tooltip>
			<n-tooltip placement="left">
				<template #trigger>
					<button class="tool-dock__button is-note" :class="{ 'is-active': activeTool === 'note' }" type="button" @click="toggle('note')">
						<Icon name="mdi:notebook-edit-outline" :size="21" />
					</button>
				</template>
				{{ noteT("floatingNote.open") }}
			</n-tooltip>
		</nav>
	</div>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import FloatingNote from "@/components/common/FloatingNote.vue"
import SystemDiagnosticPanel from "@/components/common/SystemDiagnosticPanel.vue"
import { floatingNoteMessages } from "@/i18n/locales/floatingNote"
import { systemDiagnosticMessages } from "@/i18n/locales/systemDiagnostic"
import { useAuthStore } from "@/store/auth"
import { computed, onBeforeUnmount, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"

type DockTool = "diagnostic" | "note"

const SCREEN_GAP = 16
const DOCK_HEIGHT = 144
const authStore = useAuthStore()
const { t } = useI18n({ messages: systemDiagnosticMessages })
const { t: noteT } = useI18n({ messages: floatingNoteMessages })
const activeTool = ref<DockTool | null>(null)
const positionY = ref(120)
let dragOffsetY = 0

const canDiagnose = computed(() => authStore.role === "ADMIN" || authStore.role === "SUPER")
const storageKey = computed(() => `gopanel_floating_tool_dock_${authStore.user?.id || "default"}`)
const dockStyle = computed(() => ({ transform: `translate3d(0, ${positionY.value}px, 0)` }))
const panelStyle = computed(() => ({ top: `${panelTop()}px` }))

function clampY(value: number) {
	const fallback = 120
	const safeValue = Number.isFinite(value) ? value : fallback
	return Math.min(Math.max(SCREEN_GAP, safeValue), Math.max(SCREEN_GAP, window.innerHeight - DOCK_HEIGHT - SCREEN_GAP))
}

function panelTop() {
	const estimatedHeight = activeTool.value === "diagnostic" ? 720 : 420
	return Math.min(positionY.value, Math.max(SCREEN_GAP, window.innerHeight - estimatedHeight - SCREEN_GAP))
}

function persistPosition() {
	localStorage.setItem(storageKey.value, String(positionY.value))
}

function restorePosition() {
	const saved = localStorage.getItem(storageKey.value)
	positionY.value = saved === null ? 120 : clampY(Number(saved))
}

function toggle(tool: DockTool) {
	activeTool.value = activeTool.value === tool ? null : tool
}

function startDrag(event: PointerEvent) {
	if (event.button !== 0 || window.matchMedia("(max-width: 640px)").matches) return
	dragOffsetY = event.clientY - positionY.value
	window.addEventListener("pointermove", drag)
	window.addEventListener("pointerup", stopDrag, { once: true })
}

function drag(event: PointerEvent) {
	positionY.value = clampY(event.clientY - dragOffsetY)
}

function stopDrag() {
	window.removeEventListener("pointermove", drag)
	persistPosition()
}

function handleResize() {
	positionY.value = clampY(positionY.value)
	persistPosition()
}

onMounted(() => {
	restorePosition()
	window.addEventListener("resize", handleResize)
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	window.removeEventListener("pointermove", drag)
})
</script>

<style scoped>
.tool-dock-layer { position: fixed; inset: 0; z-index: 1800; pointer-events: none; }
.tool-dock { position: absolute; top: 0; right: 0; display: flex; width: 54px; flex-direction: column; align-items: stretch; overflow: hidden; pointer-events: auto; border: 1px solid rgb(148 163 184 / 28%); border-right: 0; border-radius: 14px 0 0 14px; background: var(--n-color, var(--bg-default-color)); box-shadow: 0 12px 30px rgb(15 23 42 / 18%); }
.tool-dock__handle, .tool-dock__button { display: grid; width: 100%; place-items: center; border: 0; background: transparent; color: inherit; }
.tool-dock__handle { height: 30px; cursor: ns-resize; touch-action: none; opacity: 0.42; }
.tool-dock__button { height: 54px; cursor: pointer; border-top: 1px solid rgb(148 163 184 / 18%); transition: background 0.15s ease, color 0.15s ease; }
.tool-dock__button:hover, .tool-dock__button.is-active { background: rgb(59 130 246 / 10%); }
.tool-dock__button.is-diagnostic { color: #2563eb; }
.tool-dock__button.is-note { color: #d97706; }
.tool-dock-panel { position: absolute; right: 70px; pointer-events: auto; }
@media (max-width: 640px) {
	.tool-dock { top: auto; right: 12px; bottom: 12px; width: 52px; transform: none !important; border-right: 1px solid rgb(148 163 184 / 28%); border-radius: 14px; }
	.tool-dock__handle { display: none; }
	.tool-dock-panel { top: 16px !important; right: 16px; }
}
</style>
