<script lang="ts" setup>
interface Props {
	showFooter?: boolean
}

withDefaults(defineProps<Props>(), {
	showFooter: false
})
</script>

<template>
	<transition appear mode="out-in" name="fade-slide">
		<section class="cus-scroll-y wh-full p-15 flex-col bg-[#f5f6fb]" dark:bg-hex-121212>
			<slot />
			<app-footer v-if="showFooter" mt-auto pt-20 />
		</section>
	</transition>
</template>
