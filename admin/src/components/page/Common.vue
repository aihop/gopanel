<script lang="ts" setup>
interface Props {
	showFooter?: boolean
	showHeader?: boolean
	title?: string
}

withDefaults(defineProps<Props>(), {
	showFooter: false,
	showHeader: false,
	title: undefined
})
</script>

<template>
	<app-page :show-footer="showFooter">
		<div class="flex flex-col gap-10">
			<header v-if="showHeader">
				<slot v-if="$slots.header" name="header" />
				<n-card v-else size="small">
					<slot name="tabbar" />
				</n-card>
			</header>
			<n-card>
				<slot />
			</n-card>
		</div>
	</app-page>
</template>
