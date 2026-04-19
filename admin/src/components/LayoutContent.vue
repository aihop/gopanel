<template>
	<div class="main-box">
		<div v-if="slots.app" class="content-container__app">
			<slot name="app"></slot>
		</div>
		<div v-if="slots.toolbar" class="content-container__toolbar">
			<slot name="toolbar"></slot>
		</div>
		<div v-if="slots.search" class="content-container__search">
			<n-card>
				<slot name="search"></slot>
			</n-card>
		</div>

		<div class="content-container_form">
			<slot name="form">
				<slot name="button"></slot>
			</slot>
		</div>
		<div v-if="slots.main" class="content-container__main">
			<n-card>
				<div v-if="slots.title || title" class="content-container__title">
					<slot name="title">
						<back-button
							v-if="showBack"
							:path="backPath"
							:name="backName"
							:to="backTo"
							:header="title"
							:reload="reload"
						>
							<template v-if="slots.buttons" #buttons>
								<slot name="buttons"></slot>
							</template>
						</back-button>

						<span v-else>
							{{ title }}
							<span v-if="slots.buttons">
								<n-divider direction="vertical" />
								<slot name="buttons"></slot>
							</span>
							<span class="float-right">
								<slot v-if="slots.rightButton" name="rightButton"></slot>
							</span>
						</span>
						<div v-if="prop.divider">
							<div class="divider"></div>
						</div>
					</slot>
				</div>
				<div v-if="slots.prompt" class="prompt">
					<slot name="prompt"></slot>
				</div>
				<div class="main-content">
					<slot name="main"></slot>
				</div>
			</n-card>
		</div>
		<slot></slot>
	</div>
</template>

<script setup lang="ts">
import BackButton from "@/components/BackButton.vue"
import { computed, useSlots } from "vue"

defineOptions({ name: "LayoutContent" })
const prop = defineProps({
	title: String,
	backPath: String,
	backName: String,
	backTo: Object,
	reload: Boolean,
	divider: Boolean
})
const slots = useSlots()
const showBack = computed(() => {
	const { backPath, backName, backTo, reload } = prop
	return backPath || backName || backTo || reload
})
</script>

<style lang="scss">
.content-container__app {
	margin-top: 20px;
}

.content-container__search {
	margin-top: 20px;
	.n-card {
		--n-card-padding: 12px;
	}
}

.content-container__title {
	font-weight: 700;
	font-size: 18px;
}

.content-container__toolbar {
	margin-top: 20px;
}

.content-container_form {
	text-align: -webkit-center;
	width: 60%;
	margin-left: 15%;
	.form-button {
		float: right;
	}
}

.content-container__main {
	margin-top: 20px;
}

.prompt {
	margin-top: 10px;
}

.divider {
	margin-top: 20px;
	border: 0;
	border-top: var(--panel-border);
}

.main-box {
	position: relative;
}
.main-content {
	margin-top: 20px;
}
</style>
