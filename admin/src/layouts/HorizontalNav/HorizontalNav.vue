<template>
	<div class="layout flex flex-col">
		<HeaderBar />
		<Sidebar />
		<MainContainer class="grow">
			<slot />
		</MainContainer>
	</div>
</template>

<script lang="ts" setup>
import { useMessage } from "naive-ui"
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
