<template>
	<div class="floating-note-layer">
		<section
			v-if="visible"
			class="floating-note"
			:style="panelStyle"
		>
			<header class="floating-note__header" @pointerdown="startDrag">
				<div class="flex min-w-0 items-center gap-2">
					<Icon name="mdi:notebook-edit-outline" :size="18" />
					<span class="truncate font-medium">{{ t("floatingNote.title") }}</span>
					<span class="save-state" :class="{ 'is-dirty': dirty }">
						{{ dirty ? t("floatingNote.unsaved") : t("floatingNote.saved") }}
					</span>
				</div>
				<n-tooltip>
					<template #trigger>
						<n-button quaternary circle size="small" @pointerdown.stop @click="close">
							<template #icon><Icon name="mdi:close" :size="17" /></template>
						</n-button>
					</template>
					{{ t("floatingNote.close") }}
				</n-tooltip>
			</header>

			<div class="floating-note__body">
				<n-spin v-if="loading" class="grow" />
				<n-alert v-else-if="loadError" type="error" :show-icon="false">
					<div class="flex items-center justify-between gap-2">
						<span>{{ t("floatingNote.loadFailed") }}</span>
						<n-button text type="primary" @click="loadNote">{{ t("floatingNote.retry") }}</n-button>
					</div>
				</n-alert>
				<n-input
					v-else
					v-model:value="content"
					type="textarea"
					class="floating-note__input"
					:placeholder="t('floatingNote.placeholder')"
					maxlength="20000"
				/>
				<n-alert v-if="saveError" type="error" :show-icon="false">
					{{ t("floatingNote.saveFailed") }}
				</n-alert>
			</div>

			<footer class="floating-note__footer">
				<span class="text-xs opacity-60">{{ t("floatingNote.length", { count: content.length }) }}</span>
				<n-button type="primary" size="small" :loading="saving" :disabled="loading || !dirty" @click="saveNote">
					{{ saving ? t("floatingNote.saving") : t("floatingNote.save") }}
				</n-button>
			</footer>
		</section>

		<n-tooltip v-else placement="left">
			<template #trigger>
				<n-button class="floating-note__launcher" type="primary" circle size="large" @click="open">
					<template #icon><Icon name="mdi:notebook-edit-outline" :size="21" /></template>
				</n-button>
			</template>
			{{ t("floatingNote.open") }}
		</n-tooltip>
	</div>
</template>

<script setup lang="ts">
import Icon from "@/components/common/Icon.vue"
import { userNoteGetAPI, userNoteSaveAPI } from "@/api/modules/userNote"
import { floatingNoteMessages } from "@/i18n/locales/floatingNote"
import { useAuthStore } from "@/store/auth"
import { useMessage } from "naive-ui"
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"

const PANEL_WIDTH = 360
const PANEL_HEIGHT = 360
const SCREEN_GAP = 16

const authStore = useAuthStore()
const message = useMessage()
const { t } = useI18n({ messages: floatingNoteMessages })
const visible = ref(true)
const loading = ref(false)
const saving = ref(false)
const loadError = ref(false)
const saveError = ref(false)
const content = ref("")
const savedContent = ref("")
const position = ref({ x: SCREEN_GAP, y: 96 })
let dragOffset = { x: 0, y: 0 }

const userID = computed(() => authStore.user?.id || 0)
const storageKey = computed(() => `gopanel_floating_note_${userID.value || "default"}`)
const dirty = computed(() => content.value !== savedContent.value)
const panelWidth = () => Math.min(PANEL_WIDTH, window.innerWidth - SCREEN_GAP * 2)
const panelHeight = () => Math.min(PANEL_HEIGHT, window.innerHeight - SCREEN_GAP * 2)
const panelStyle = computed(() => ({
	width: `${PANEL_WIDTH}px`,
	height: `${panelHeight()}px`,
	transform: `translate3d(${position.value.x}px, ${position.value.y}px, 0)`
}))

function clampPosition(x: number, y: number) {
	const safeX = Number.isFinite(x) ? x : window.innerWidth - panelWidth() - 24
	const safeY = Number.isFinite(y) ? y : 96
	return {
		x: Math.min(Math.max(SCREEN_GAP, safeX), Math.max(SCREEN_GAP, window.innerWidth - panelWidth() - SCREEN_GAP)),
		y: Math.min(Math.max(SCREEN_GAP, safeY), Math.max(SCREEN_GAP, window.innerHeight - panelHeight() - SCREEN_GAP))
	}
}

function persistWindowState() {
	localStorage.setItem(storageKey.value, JSON.stringify({ visible: visible.value, ...position.value }))
}

function restoreWindowState() {
	position.value = clampPosition(window.innerWidth - PANEL_WIDTH - 24, 96)
	try {
		const state = JSON.parse(localStorage.getItem(storageKey.value) || "null")
		if (state && typeof state === "object") {
			visible.value = state.visible !== false
			position.value = clampPosition(Number(state.x), Number(state.y))
		}
	} catch {
		localStorage.removeItem(storageKey.value)
	}
}

async function loadNote() {
	loading.value = true
	loadError.value = false
	try {
		const response = await userNoteGetAPI()
		content.value = response.data?.content || ""
		savedContent.value = content.value
	} catch {
		loadError.value = true
	} finally {
		loading.value = false
	}
}

async function saveNote() {
	if (!dirty.value || saving.value) return
	const contentToSave = content.value
	saving.value = true
	saveError.value = false
	try {
		const response = await userNoteSaveAPI(contentToSave)
		savedContent.value = response.data.content
		message.success(t("floatingNote.saveSuccess"))
	} catch {
		saveError.value = true
	} finally {
		saving.value = false
	}
}

function open() {
	visible.value = true
	persistWindowState()
	nextTick(() => {
		position.value = clampPosition(position.value.x, position.value.y)
	})
}

function close() {
	visible.value = false
	persistWindowState()
}

function startDrag(event: PointerEvent) {
	if (event.button !== 0) return
	dragOffset = { x: event.clientX - position.value.x, y: event.clientY - position.value.y }
	window.addEventListener("pointermove", drag)
	window.addEventListener("pointerup", stopDrag, { once: true })
}

function drag(event: PointerEvent) {
	position.value = clampPosition(event.clientX - dragOffset.x, event.clientY - dragOffset.y)
}

function stopDrag() {
	window.removeEventListener("pointermove", drag)
	persistWindowState()
}

function handleResize() {
	position.value = clampPosition(position.value.x, position.value.y)
	persistWindowState()
}

onMounted(() => {
	restoreWindowState()
	loadNote()
	window.addEventListener("resize", handleResize)
})

onBeforeUnmount(() => {
	window.removeEventListener("resize", handleResize)
	window.removeEventListener("pointermove", drag)
})
</script>

<style scoped lang="scss">
.floating-note-layer {
	position: fixed;
	inset: 0;
	z-index: 1800;
	pointer-events: none;
}

.floating-note {
	position: absolute;
	top: 0;
	left: 0;
	display: flex;
	flex-direction: column;
	overflow: hidden;
	pointer-events: auto;
	border: 1px solid rgb(245 158 11 / 30%);
	border-radius: 16px;
	background: var(--n-color, var(--bg-default-color));
	box-shadow: 0 20px 48px rgb(15 23 42 / 22%);
}

.floating-note__header {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	padding: 10px 10px 10px 14px;
	cursor: move;
	touch-action: none;
	user-select: none;
	border-bottom: 1px solid rgb(245 158 11 / 22%);
	background: rgb(254 243 199 / 86%);
	color: #78350f;
}

.save-state {
	font-size: 11px;
	opacity: 0.65;

	&.is-dirty {
		color: #b45309;
		opacity: 1;
	}
}

.floating-note__body {
	display: flex;
	min-height: 0;
	flex: 1;
	flex-direction: column;
	gap: 8px;
	padding: 12px;
}

.floating-note__input {
	flex: 1;
	min-height: 0;

	:deep(textarea) {
		height: 100% !important;
		resize: none;
		line-height: 1.65;
	}

	:deep(.n-input-wrapper),
	:deep(.n-input__textarea) {
		height: 100%;
	}
}

.floating-note__footer {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	padding: 10px 12px;
	border-top: 1px solid var(--border-color);
}

.floating-note__launcher {
	position: absolute;
	right: 22px;
	bottom: 22px;
	pointer-events: auto;
	box-shadow: 0 12px 28px rgb(15 23 42 / 22%);
}

@media (max-width: 480px) {
	.floating-note {
		max-width: calc(100vw - 32px);
	}
}
</style>
