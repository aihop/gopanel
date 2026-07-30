<template>
  <n-modal
    :show="mergedShow"
    :mask-closable="maskClosable"
    :close-on-esc="closeOnEsc"
    :trap-focus="trapFocus"
    :display-directive="displayDirective"
    :auto-focus="autoFocus"
    :block-scroll="blockScroll"
    :to="to"
    class="full-modal-shell"
    :class="{ 'is-fullscreen': currentFullscreen }"
    @update:show="(value) => (mergedShow = value)"
    @after-enter="$emit('after-enter')"
    @after-leave="$emit('after-leave')"
    @close="handleClose"
  >
    <div
      class="full-modal__backdrop"
      @click.stop
    >
      <div
        class="full-modal"
        :style="panelStyle"
      >
        <div
          v-if="showHeader"
          class="full-modal__header"
        >
          <div class="full-modal__header-main">
            <div class="full-modal__title-wrap">
              <slot name="title">
                <div class="full-modal__title">{{ title }}</div>
              </slot>
              <div
                v-if="subtitle || $slots.subtitle"
                class="full-modal__subtitle"
              >
                <slot name="subtitle">
                  {{ subtitle }}
                </slot>
              </div>
            </div>
            <div
              v-if="$slots.headerExtra"
              class="full-modal__header-extra"
            >
              <slot name="headerExtra"></slot>
            </div>
          </div>

          <div
            v-if="$slots.toolbar"
            class="full-modal__toolbar"
          >
            <slot name="toolbar"></slot>
          </div>

          <n-button
            v-if="showFullscreenToggle"
            quaternary
            circle
            class="full-modal__fullscreen"
            @click="toggleFullscreen"
          >
            <Icon :name="currentFullscreen ? 'mdi:fullscreen-exit' : 'mdi:fullscreen'" />
          </n-button>

          <n-button
            v-if="closable"
            quaternary
            circle
            class="full-modal__close"
            @click="close"
          >
            <span class="full-modal__close-icon">x</span>
          </n-button>
        </div>

        <div
          class="full-modal__body"
          :class="bodyClass"
        >
          <n-scrollbar
            v-if="bodyScrollable"
            class="full-modal__scrollbar"
            :x-scrollable="xScrollable"
          >
            <div
              class="full-modal__content"
              :style="contentStyle"
            >
              <slot></slot>
            </div>
          </n-scrollbar>
          <div
            v-else
            class="full-modal__content full-modal__content--fill"
            :style="contentStyle"
          >
            <slot></slot>
          </div>
        </div>

        <div
          v-if="showFooter || $slots.footer"
          class="full-modal__footer"
        >
          <slot name="footer">
            <n-space justify="end">
              <n-button
                v-if="showCancel"
                @click="close"
              >
                {{ cancelText }}
              </n-button>
              <n-button
                v-if="showConfirm"
                type="primary"
                :loading="confirmLoading"
                @click="handleConfirm"
              >
                {{ confirmText }}
              </n-button>
            </n-space>
          </slot>
        </div>
      </div>
    </div>
  </n-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { NButton, NModal, NScrollbar, NSpace } from "naive-ui"

defineOptions({ name: "FullModal" })

const props = withDefaults(defineProps<{
	show: boolean
	title?: string
	subtitle?: string
	fullscreen?: boolean
	showFullscreenToggle?: boolean
	width?: string
	height?: string
	maxWidth?: string
	maxHeight?: string
	bodyPadding?: string
	bodyScrollable?: boolean
	maskClosable?: boolean
	closeOnEsc?: boolean
	closable?: boolean
	showHeader?: boolean
	showFooter?: boolean
	showCancel?: boolean
	showConfirm?: boolean
	confirmText?: string
	cancelText?: string
	confirmLoading?: boolean
	xScrollable?: boolean
	trapFocus?: boolean
	autoFocus?: boolean
	blockScroll?: boolean
	displayDirective?: "if" | "show"
	to?: string | HTMLElement
	bodyClass?: string
}>(), {
	title: "",
	subtitle: "",
	fullscreen: false,
	showFullscreenToggle: true,
	width: "min(1200px, calc(100vw - 48px))",
	height: "min(860px, calc(100vh - 48px))",
	maxWidth: "100vw",
	maxHeight: "100vh",
	bodyPadding: "20px 20px 20px",
	bodyScrollable: true,
	maskClosable: false,
	closeOnEsc: true,
	closable: true,
	showHeader: true,
	showFooter: false,
	showCancel: true,
	showConfirm: true,
	confirmText: "确定",
	cancelText: "取消",
	confirmLoading: false,
	xScrollable: false,
	trapFocus: true,
	autoFocus: true,
	blockScroll: true,
	displayDirective: "if",
	to: "body",
	bodyClass: ""
})

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "close"): void
	(e: "confirm"): void
	(e: "after-enter"): void
	(e: "after-leave"): void
}>()

const mergedShow = computed({
	get: () => props.show,
	set: value => emit("update:show", value)
})

const currentFullscreen = ref(props.fullscreen)

watch(
	() => props.fullscreen,
	value => {
		currentFullscreen.value = value
	}
)

const panelStyle = computed(() => {
	if (currentFullscreen.value) {
		return {
			width: "100vw",
			height: "100vh",
			maxWidth: "100vw",
			maxHeight: "100vh"
		}
	}
	return {
		width: props.width,
		height: props.height,
		maxWidth: props.maxWidth,
		maxHeight: props.maxHeight
	}
})

const contentStyle = computed(() => ({
	padding: props.bodyPadding
}))

function open() {
	mergedShow.value = true
}

function close() {
	mergedShow.value = false
	emit("close")
}

function toggleFullscreen() {
	currentFullscreen.value = !currentFullscreen.value
}

function toggle() {
	if (mergedShow.value) {
		close()
		return
	}
	open()
}

function handleClose() {
	emit("close")
}

function handleConfirm() {
	emit("confirm")
}

defineExpose({
	open,
	close,
	toggle
})
</script>

<style scoped lang="scss">
.full-modal-shell {
	:deep(.n-modal-mask) {
		backdrop-filter: blur(4px);
		background: rgba(15, 23, 42, 0.35);
	}

	:deep(.n-modal) {
		width: auto;
		margin: 0;
		padding: 0;
    background: transparent;
    box-shadow: none;
	}
}

.full-modal__backdrop {
	display: flex;
	align-items: center;
	justify-content: center;
	box-sizing: border-box;
}

/* 全屏模式下，移除 backdrop 的 padding，让模态框完全铺满 100vw 和 100vh */
.full-modal-shell.is-fullscreen .full-modal__backdrop {
	padding: 0;
  width: 100vw;
  height: 100vh;
}

.full-modal {
	position: relative;
	display: flex;
	flex-direction: column;
	overflow: hidden;
	background: var(--n-color);
	color: var(--n-text-color);
	border: 1px solid var(--n-border-color);
	border-radius: 24px;
	box-shadow: 0 24px 64px rgba(15, 23, 42, 0.18);
  /* 确保非全屏时，遮罩和弹出层内容紧贴在一起，避免点击空白处异常 */
  max-width: 100%;
  max-height: 100%;
}

/* 全屏模式下，移除圆角和边框，铺满整个屏幕 */
.full-modal-shell.is-fullscreen .full-modal {
	border: none;
	border-radius: 0;
	box-shadow: none;
  width: 100vw !important;
  height: 100vh !important;
  max-width: 100vw !important;
  max-height: 100vh !important;
}

.full-modal__header {
	position: relative;
	flex-shrink: 0;
	padding: 22px 72px 18px 24px;
	border-bottom: 1px solid var(--n-border-color);
	background: var(--n-color);
}

.full-modal__header-main {
	display: flex;
	align-items: flex-start;
	justify-content: space-between;
	gap: 16px;
}

.full-modal__title-wrap {
	min-width: 0;
}

.full-modal__title {
	font-size: 20px;
	font-weight: 700;
	line-height: 1.3;
	color: var(--n-text-color);
}

.full-modal__subtitle {
	margin-top: 6px;
	font-size: 13px;
	line-height: 1.6;
	color: var(--n-text-color-3);
}

.full-modal__header-extra {
	flex-shrink: 0;
}

.full-modal__toolbar {
	margin-top: 16px;
}

.full-modal__close {
	position: absolute;
	top: 18px;
	right: 20px;
}

.full-modal__fullscreen {
	position: absolute;
	top: 18px;
	right: 64px;
}

.full-modal__close-icon {
	font-size: 18px;
	line-height: 1;
}

.full-modal__body {
	flex: 1;
	min-height: 0;
}

.full-modal__scrollbar {
	height: 100%;
}

.full-modal__content {
	min-height: 100%;
	box-sizing: border-box;
}

.full-modal__content--fill {
	height: 100%;
	min-height: 0;
	overflow: hidden;
}

.full-modal__footer {
	flex-shrink: 0;
	padding: 16px 24px 20px;
	border-top: 1px solid var(--n-border-color);
	background: var(--n-color);
}
</style>
