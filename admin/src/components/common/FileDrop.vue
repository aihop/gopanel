<template>
	<form class="file-drop">
		<label>
			<slot />
			<input type="file" :multiple="multiple" :accept @change="onChange" />
		</label>
	</form>
</template>

<script lang="ts" setup>
const { accept = ".jpg,.jpeg,.png,.webp", multiple = false } = defineProps<{
	accept: string
	multiple: boolean
}>()

const emit = defineEmits<{
	(e: "change-file", value: FileList | File | null): void
}>()

function onChange(event: Event) {
	const input = event.target as HTMLInputElement
	if (multiple) {
		emit("change-file", input?.files)
	} else {
		if (input?.files?.length) {
			emit("change-file", input?.files[0])
		}
	}
	input?.form?.reset()
}
</script>

<style lang="scss" scoped>
.file-drop {
	label {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 100%;
		height: 100%;
		cursor: pointer;

		input {
			display: none;
		}
	}
}
</style>
