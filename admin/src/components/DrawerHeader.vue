<template>
	<n-page-header @on-back="props.back">
		<template #title>
			<span>{{ header }}</span>
			<span v-if="resource && !hideResource">
				-
				<n-tooltip v-if="resource.length > 25" placement="bottom">
					<template #trigger>
						<n-tag type="success">{{ `${resource.substring(0, 23)}...` }}</n-tag>
					</template>
					{{ resource }}
				</n-tooltip>
				<n-tag v-else type="success">{{ resource }}</n-tag>
			</span>
			<n-divider v-if="slots.buttons" vertical />
			<slot v-if="slots.buttons" name="buttons"></slot>
		</template>
		<template #extra>
			<slot v-if="slots.extra" name="extra"></slot>
		</template>
	</n-page-header>
</template>

<script lang="ts" setup>
import { NDivider, NPageHeader, NTag, NTooltip } from "naive-ui"
import { useSlots } from "vue"

defineOptions({ name: "DrawerHeader" })

const props = defineProps<{
	header?: string
	back?: () => void
	resource?: string
	hideResource?: boolean
}>()

const slots = useSlots()
</script>
