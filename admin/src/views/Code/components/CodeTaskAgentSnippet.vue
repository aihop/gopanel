<script setup lang="ts">
import type { CodeTaskListItem } from "@/api/interface/codeTasks"
import Icon from "@/components/common/Icon.vue"

defineProps<{ task: CodeTaskListItem }>()
</script>

<template>
  <!--
		执行器最后说的那句话 —— 也就是「终端里现在是什么情况」。
		内容来自 ai_messages（对话已由 code_native_history 固化进库），
		所以不用碰终端、不用解析 codex 的 rollout 文件，后端一次批量 SQL 就取到，已截断到 160 字。

		只给选中行用：同时只有一条，不会把整个列表撑松。
	-->
  <div
    v-if="task.summary.lastAgentMessage"
    class="agent-snippet mt-2 flex items-start gap-1.5 rounded-lg px-2 py-1.5 text-[11px] leading-relaxed"
    :title="task.summary.lastAgentMessage"
  >
    <Icon
      name="mdi:robot-outline"
      :size="12"
      class="mt-0.5 shrink-0 opacity-70"
    />
    <span class="line-clamp-2 min-w-0">{{ task.summary.lastAgentMessage }}</span>
  </div>
</template>

<style scoped>
/*
	用 --fg-tertiary-color / --border-color 这些全局 token，不用 naive-ui 的 --n-*：
	后者是按组件注入的，普通元素上解析不到，整条声明会失效（焦点竖线就栽在这上面）。
*/
.agent-snippet {
	color: var(--fg-tertiary-color);
	background: color-mix(in srgb, var(--border-color) 28%, transparent);
}
</style>
