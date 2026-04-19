<template>
	<div class="page">
		<div class="spacer">Full Width</div>
	</div>
</template>

<style lang="scss" scoped>
.spacer {
	background: rgba(var(--border-color-rgb) / 0.5);
	background: repeating-linear-gradient(
		-45deg,
		rgba(var(--border-color-rgb) / 0.5),
		rgba(var(--border-color-rgb) / 0.5) 1px,
		transparent 1px,
		transparent 20px
	);
	width: 100%;
	height: 200vh;
	border-radius: 14px;
	border: 4px dashed rgba(var(--border-color-rgb) / 0.5);
	padding: 30px;
}
</style>
