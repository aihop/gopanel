<script setup lang="ts">
import { useI18n } from "vue-i18n"
import Icon from "@/components/common/Icon.vue"
import { codeTerminalMessages } from "../codeTerminalMessages"

const { t } = useI18n({ messages: codeTerminalMessages })

defineProps<{ visible: boolean }>()
defineEmits<{ jump: [] }>()
</script>

<template>
  <!--
		翻在历史里时才出现。悬浮在右下角而不是占一行工具条：终端的每一行都是内容，
		常驻控件会一直吃掉可视高度。贴底后自动消失，免得挡住最新输出。
	-->
  <Transition name="scroll-anchor">
    <button
      v-if="visible"
      class="scroll-anchor"
      type="button"
      :title="t('code.terminalJumpToBottom')"
      :aria-label="t('code.terminalJumpToBottom')"
      @click="$emit('jump')"
    >
      <Icon
        name="carbon:arrow-down"
        :size="16"
      />
    </button>
  </Transition>
</template>

<style scoped>
.scroll-anchor {
	position: absolute;
	right: 16px;
	bottom: 16px;
	z-index: 3;
	display: flex;
	align-items: center;
	justify-content: center;
	width: 32px;
	height: 32px;
	border: 1px solid rgb(255 255 255 / 18%);
	border-radius: 50%;
	/* 终端底色是写死的 #1e1e1e，这里跟着它走，不用主题变量。 */
	background: rgb(45 45 45 / 92%);
	color: #d4d4d4;
	cursor: pointer;
	transition:
		background 0.15s,
		transform 0.15s;
}

.scroll-anchor:hover {
	background: rgb(64 64 64 / 96%);
	transform: translateY(-1px);
}

.scroll-anchor-enter-active,
.scroll-anchor-leave-active {
	transition:
		opacity 0.15s,
		transform 0.15s;
}

.scroll-anchor-enter-from,
.scroll-anchor-leave-to {
	opacity: 0;
	transform: translateY(4px);
}
</style>
