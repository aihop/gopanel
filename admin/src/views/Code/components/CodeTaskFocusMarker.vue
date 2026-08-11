<script setup lang="ts">
defineProps<{ active?: boolean }>()
</script>

<template>
  <!--
		任务行「当前选中」的竖线标识，工作台侧栏和开发面板共用。

		做成 flex 里的真实元素，不用 ::before 贴 left:0 ——
		任务行是圆角卡片，伪元素会被圆角吞掉，看着像毛边而不是标识。
		未选中时保留同宽度的透明条，切换选中不会让整行文字左右跳。
	-->
  <span
    class="code-task-focus-marker w-[3px] shrink-0 self-stretch rounded-full"
    :class="active ? 'code-task-focus-marker--on' : ''"
  />
</template>

<style scoped>
.code-task-focus-marker {
	background: transparent;
	transition:
		background-color 0.18s ease,
		box-shadow 0.18s ease;
}

.code-task-focus-marker--on {
	background: var(--primary-color);
	box-shadow: 0 0 8px color-mix(in srgb, var(--primary-color) 55%, transparent);
}
</style>
