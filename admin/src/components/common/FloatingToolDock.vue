<template>
	<div class="tool-dock-layer">
		<div ref="panelRef" v-show="activeTool" class="tool-dock-panel" :style="panelStyle">
			<keep-alive>
				<SystemDiagnosticPanel v-if="activeTool === 'diagnostic'" @close="activeTool = null" @drag-start="startDrag" />
				<FloatingNote v-else-if="activeTool === 'note'" @close="activeTool = null" @drag-start="startDrag" />
			</keep-alive>
		</div>

		<nav ref="dockRef" class="tool-dock" :style="dockStyle">
			<n-tooltip placement="left">
				<template #trigger>
					<button class="tool-dock__handle" type="button" :aria-label="collapsed ? t('systemDiagnostic.expand') : t('systemDiagnostic.drag')" @pointerdown="startDrag" @click="toggleCollapse">
						<Icon :name="collapsed ? 'mdi:chevron-right' : 'mdi:drag-vertical'" :size="16" />
					</button>
				</template>
				{{ collapsed ? t("systemDiagnostic.expand") : t("systemDiagnostic.drag") }}
			</n-tooltip>
			<template v-if="!collapsed">
				<n-tooltip v-if="canDiagnose" placement="left">
					<template #trigger>
						<button ref="diagnosticButtonRef" class="tool-dock__button is-diagnostic" :class="{ 'is-active': activeTool === 'diagnostic' }" type="button" @click="toggle('diagnostic')">
							<Icon name="mdi:stethoscope" :size="18" />
						</button>
					</template>
					{{ t("systemDiagnostic.open") }}
				</n-tooltip>
				<n-tooltip placement="left">
					<template #trigger>
						<button ref="noteButtonRef" class="tool-dock__button is-note" :class="{ 'is-active': activeTool === 'note' }" type="button" @click="toggle('note')">
							<Icon name="mdi:notebook-edit-outline" :size="17" />
						</button>
					</template>
					{{ t("floatingNote.open") }}
				</n-tooltip>
				<n-tooltip v-if="showCodeImmersive" placement="left">
					<template #trigger>
						<button
							class="tool-dock__button is-code"
							:class="{ 'is-active': codeImmersiveStore.isImmersive }"
							type="button"
							:aria-label="codeImmersiveLabel"
							@click="codeImmersiveStore.toggleImmersive()"
						>
							<Icon
								:name="codeImmersiveStore.isImmersive ? 'fluent:full-screen-minimize-24-regular' : 'fluent:full-screen-maximize-24-regular'"
								:size="17"
							/>
						</button>
					</template>
					{{ codeImmersiveLabel }}
				</n-tooltip>
			</template>
		</nav>
	</div>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import FloatingNote from "@/components/common/FloatingNote.vue"
import SystemDiagnosticPanel from "@/components/common/SystemDiagnosticPanel.vue"
import { floatingNoteMessages } from "@/i18n/locales/floatingNote"
import { systemDiagnosticMessages } from "@/i18n/locales/systemDiagnostic"
import { codeImmersiveMessages } from "@/i18n/locales/codeImmersive"
import { useAuthStore } from "@/store/auth"
import { useCodeImmersiveStore } from "@/store/codeImmersive"
import { useMediaQuery } from "@vueuse/core"
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute } from "vue-router"

type DockTool = "diagnostic" | "note"

const SCREEN_GAP = 16
const DOCK_WIDTH = 40
const BUTTON_HEIGHT = 40
const COLLAPSED_WIDTH = 32
const authStore = useAuthStore()
const codeImmersiveStore = useCodeImmersiveStore()
const route = useRoute()
const isDesktop = useMediaQuery("(min-width: 1000px)")
const { t } = useI18n({
	messages: {
		zh: { ...systemDiagnosticMessages.zh, ...floatingNoteMessages.zh, ...codeImmersiveMessages.zh },
		en: { ...systemDiagnosticMessages.en, ...floatingNoteMessages.en, ...codeImmersiveMessages.en }
	}
})
const activeTool = ref<DockTool | null>(null)
const collapsed = ref(false)
const positionY = ref(120)
const panelTop = ref(SCREEN_GAP)
const panelRef = ref<HTMLElement | null>(null)
const dockRef = ref<HTMLElement | null>(null)
const diagnosticButtonRef = ref<HTMLButtonElement | null>(null)
const noteButtonRef = ref<HTMLButtonElement | null>(null)
let dragOffsetY = 0

const canDiagnose = computed(() => authStore.role === "ADMIN" || authStore.role === "SUPER")
const showCodeImmersive = computed(
	() => (route.name === "Code-Index" || route.name === "Code-Project") && isDesktop.value
)
const codeImmersiveLabel = computed(() =>
	t(codeImmersiveStore.isImmersive ? "codeImmersive.exit" : "codeImmersive.enter")
)
const storageKey = computed(() => `gopanel_floating_tool_dock_${authStore.user?.id || "default"}`)
const dockStyle = computed(() => ({
	transform: `translate3d(0, ${positionY.value}px, 0)`,
	width: `${collapsed.value ? COLLAPSED_WIDTH : DOCK_WIDTH}px`
}))
const panelStyle = computed(() => ({
	top: `${panelTop.value}px`,
	right: `${collapsed.value ? COLLAPSED_WIDTH + 12 : DOCK_WIDTH + 12}px`
}))

function clampY(value: number) {
	const fallback = 120
	const safeValue = Number.isFinite(value) ? value : fallback
	const dockHeight = collapsed.value ? COLLAPSED_WIDTH + 32 : dockRef.value?.offsetHeight ?? (BUTTON_HEIGHT + 32)
	return Math.min(Math.max(SCREEN_GAP, safeValue), Math.max(SCREEN_GAP, window.innerHeight - dockHeight - SCREEN_GAP))
}

function updatePanelPosition() {
	if (!activeTool.value || !panelRef.value) return
	if (window.matchMedia("(max-width: 640px)").matches) {
		panelTop.value = SCREEN_GAP
		return
	}
	const button = activeTool.value === "diagnostic" ? diagnosticButtonRef.value : noteButtonRef.value
	if (!button) return
	const anchorY = positionY.value + button.offsetTop + button.offsetHeight / 2
	const centeredTop = anchorY - panelRef.value.offsetHeight / 2
	panelTop.value = Math.min(
		Math.max(SCREEN_GAP, centeredTop),
		Math.max(SCREEN_GAP, window.innerHeight - panelRef.value.offsetHeight - SCREEN_GAP)
	)
}

function persistPosition() {
	localStorage.setItem(storageKey.value, JSON.stringify({ y: positionY.value, collapsed: collapsed.value }))
}

function restorePosition() {
	const saved = localStorage.getItem(storageKey.value)
	if (saved) {
		try {
			const data = JSON.parse(saved)
			positionY.value = clampY(data.y ?? 120)
			collapsed.value = !!data.collapsed
		} catch {
			positionY.value = 120
		}
	} else {
		positionY.value = 120
	}
}

function toggle(tool: DockTool) {
	activeTool.value = activeTool.value === tool ? null : tool
}

function toggleCollapse() {
	collapsed.value = !collapsed.value
	if (collapsed.value) activeTool.value = null
	persistPosition()
}

function startDrag(event: PointerEvent) {
	if (event.button !== 0 || window.matchMedia("(max-width: 640px)").matches || collapsed.value) return
	event.preventDefault()
	dragOffsetY = event.clientY - positionY.value
	window.addEventListener("pointermove", drag)
	window.addEventListener("pointerup", stopDrag)
	window.addEventListener("pointercancel", stopDrag)
}

function drag(event: PointerEvent) {
	positionY.value = clampY(event.clientY - dragOffsetY)
	updatePanelPosition()
}

function stopDrag() {
	window.removeEventListener("pointermove", drag)
	window.removeEventListener("pointerup", stopDrag)
	window.removeEventListener("pointercancel", stopDrag)
	persistPosition()
}

function handleResize() {
	positionY.value = clampY(positionY.value)
	updatePanelPosition()
	persistPosition()
}

watch(activeTool, async () => {
	await nextTick()
	updatePanelPosition()
})

watch(
	() => authStore.user?.id ?? null,
	userId => codeImmersiveStore.restoreForUser(userId),
	{ immediate: true }
)

watch(collapsed, (newVal) => {
	if (newVal) activeTool.value = null
})

watch(showCodeImmersive, async () => {
	await nextTick()
	handleResize()
})

onMounted(() => {
	restorePosition()
	window.addEventListener("resize", handleResize)
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	stopDrag()
})
</script>

<style scoped>
.tool-dock-layer { position: fixed; inset: 0; z-index: 1800; pointer-events: none; }
.tool-dock { position: absolute; top: 0; right: 0; display: flex; flex-direction: column; align-items: stretch; overflow: hidden; pointer-events: auto; border: 1px solid rgb(148 163 184 / 28%); border-right: 0; border-radius: 10px 0 0 10px; background: var(--n-color, var(--bg-default-color)); box-shadow: 0 8px 20px rgb(15 23 42 / 15%); transition: width 0.2s ease; }
.tool-dock__handle, .tool-dock__button { display: grid; width: 100%; place-items: center; border: 0; background: transparent; color: inherit; }
.tool-dock__handle { height: 32px; cursor: ns-resize; touch-action: none; opacity: 0.5; transition: opacity 0.15s ease; }
.tool-dock__handle:hover { opacity: 0.8; }
.tool-dock__button { height: 40px; cursor: pointer; border-top: 1px solid rgb(148 163 184 / 18%); transition: background 0.15s ease, color 0.15s ease; }
.tool-dock__button:hover, .tool-dock__button.is-active { background: rgb(59 130 246 / 10%); }
.tool-dock__button.is-diagnostic { color: #2563eb; }
.tool-dock__button.is-note { color: #d97706; }
.tool-dock__button.is-code { color: #7c3aed; }
.tool-dock-panel { position: absolute; pointer-events: auto; transition: right 0.2s ease; }
@media (max-width: 640px) {
	.tool-dock { top: auto; right: 12px; bottom: 12px; width: 36px !important; transform: none !important; border-right: 1px solid rgb(148 163 184 / 28%); border-radius: 10px; }
	.tool-dock__handle { display: none; }
	.tool-dock-panel { top: 16px !important; right: 16px; }
}
</style>
