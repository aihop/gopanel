<template>
	<div class="page page-wrapped page-mobile-full page-without-footer flex flex-col">
		<SegmentedPage enable-resize sidebar-position="right">
			<template #main-toolbar>Main toolbar</template>
			<template #main-content>
				<div>Main content scrollable</div>
				<div class="spacer mt-5" />
			</template>
			<template #main-footer>Main footer</template>
			<template #sidebar-header>Sidebar header</template>
			<template #sidebar-content>
				<div>Sidebar content scrollable</div>
				<div class="spacer mt-5" />
			</template>
			<template #sidebar-footer>Sidebar footer</template>
		</SegmentedPage>
	</div>
</template>

<script setup lang="ts">
import SegmentedPage from "@/components/common/SegmentedPage.vue"
import { useHideLayoutFooter } from "@/composables/useHideLayoutFooter"

// :has() CSS relational pseudo-class not yet supported by Firefox
// (https://caniuse.com/css-has)
// at the moment this worker around permit to hide Layout Footer
useHideLayoutFooter()
</script>

<style lang="scss" scoped>
.page {
	.spacer {
		background: var(--border-color);
		background: repeating-linear-gradient(
			-45deg,
			var(--border-color),
			var(--border-color) 1px,
			transparent 1px,
			transparent 20px
		);
		width: 100%;
		height: 200vh;
		border-radius: 14px;
		border: 4px dashed var(--border-color);
		opacity: 0.5;
	}
}
</style>
