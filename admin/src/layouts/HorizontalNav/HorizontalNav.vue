<template>
	<div class="layout flex flex-col">
		<HeaderBar />
		<Sidebar />
		<div class="min-h-0 min-w-0 flex grow overflow-hidden">
			<MainContainer class="grow">
				<slot />
			</MainContainer>
		</div>
		<NodeDrawer />
	</div>
</template>

<script lang="ts" setup>
import { useMessage } from "naive-ui"
import NodeDrawer from "../common/NodeRail/NodeDrawer.vue"
import HeaderBar from "./HeaderBar.vue"
import MainContainer from "./MainContainer.vue"
import Sidebar from "./Sidebar.vue"
import "./main.scss"

window.$message = useMessage()
</script>

<style lang="scss" scoped>
.layout {
	width: 100%;
	height: 100%;
	height: 100svh;
	min-width: 0;
	min-height: 0;
	overflow: hidden;
	perspective: 1000px;
}
</style>
