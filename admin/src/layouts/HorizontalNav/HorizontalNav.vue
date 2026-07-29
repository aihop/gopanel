<template>
	<div class="layout flex flex-col">
		<HeaderBar />
		<Sidebar />
		<div class="flex grow overflow-hidden">
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
	width: 100vw;
	height: 100vh;
	height: 100svh;
	overflow: hidden;
	perspective: 1000px;
}
</style>
