<template>
	<button class="search-btn" @click="openBox">
		<Icon :name="SearchIcon" :size="16" />
	</button>
</template>

<script lang="ts" setup>
import Icon from "@/components/common/Icon.vue"
import { useSearchDialog } from "@/composables/useSearchDialog"

const SearchIcon = "ion:search-outline"

function openBox() {
	useSearchDialog().open()
}
</script>

<style lang="scss" scoped>
.search-btn {
	display: flex;
	align-items: center;
	justify-content: center;
	width: 32px;
	height: 32px;
	padding: 0;
	background: transparent;
	border: none;
	outline: none;
	cursor: pointer;

	:deep() {
		& > .n-icon {
			opacity: 0.5;
			transition: opacity 0.3s;
		}
	}

	&:hover {
		:deep() {
			& > .n-icon {
				opacity: 0.9;
			}
		}
	}
}
</style>
