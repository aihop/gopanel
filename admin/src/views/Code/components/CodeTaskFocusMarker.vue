<script setup lang="ts">
defineProps<{ active?: boolean }>()
</script>

<template>
  <span
    class="code-task-focus-marker w-[2px] shrink-0 self-stretch rounded-full"
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
}
</style>
