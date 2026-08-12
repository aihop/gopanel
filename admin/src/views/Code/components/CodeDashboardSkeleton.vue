<script setup lang="ts">
defineProps<{ listCollapsed: boolean }>()
</script>

<template>
	<div class="flex min-h-0 flex-1 flex-col" aria-hidden="true">
		<div class="mb-4 flex flex-wrap items-center gap-3 py-1">
			<n-skeleton text style="width: 92px" />
			<n-skeleton text style="width: 72px" />
			<div class="order-last flex w-full items-center gap-2 md:order-none md:w-auto md:flex-1">
				<n-skeleton
					v-for="width in [106, 112]"
					:key="width"
					height="32px"
					:width="width"
					:sharp="false"
				/>
			</div>
			<n-skeleton circle size="medium" />
			<n-skeleton class="hidden sm:block" height="30px" width="82px" :sharp="false" />
		</div>

		<div
			class="grid min-h-0 flex-1 gap-5"
			:class="
				listCollapsed
					? 'grid-cols-1 grid-rows-1'
					: 'grid-cols-1 grid-rows-[minmax(220px,2fr)_minmax(260px,3fr)] xl:grid-cols-[400px_minmax(0,1fr)] xl:grid-rows-1'
			"
		>
			<section v-if="!listCollapsed" class="skeleton-panel min-h-0 overflow-hidden rounded-[24px] p-4">
				<div v-for="row in 3" :key="row" class="flex items-start gap-3 px-2 py-5">
					<n-skeleton height="34px" width="3px" :sharp="false" />
					<div class="min-w-0 flex-1 space-y-3">
						<div class="flex items-center gap-2">
							<n-skeleton text style="width: 62px" />
							<n-skeleton text :style="{ width: `${36 + row * 10}%` }" />
						</div>
						<div class="flex items-center gap-2">
							<n-skeleton text style="width: 48px" />
							<n-skeleton text style="width: 78px" />
						</div>
					</div>
				</div>
			</section>

			<section class="skeleton-panel flex min-h-0 flex-col overflow-hidden rounded-[24px]">
				<div class="flex h-10 shrink-0 items-center gap-2 border-b border-[var(--n-border-color)] px-4">
					<n-skeleton circle size="small" />
					<n-skeleton text style="width: 88px" />
					<div class="flex-1" />
				</div>
				<div class="min-h-0 flex-1 space-y-5 p-6">
					<n-skeleton v-for="line in 5" :key="line" text :style="{ width: `${32 + line * 9}%` }" />
				</div>
			</section>
		</div>
	</div>
</template>

<style scoped>
.skeleton-panel {
	background: color-mix(in srgb, var(--n-color) 97%, transparent);
	border: 1px solid color-mix(in srgb, var(--n-border-color) 92%, transparent);
	box-shadow: 0 8px 24px rgb(15 23 42 / 4.5%);
}
</style>
