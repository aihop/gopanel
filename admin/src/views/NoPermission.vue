<template>
	<div class="layout flex flex-col">
		<PublicToolbar :boxed="false"></PublicToolbar>
		<n-result class="mt-24" title="温馨提示">
			<template #footer>
				<div class="flex flex-col items-center">
					<p class="mb-4">{{ "您还无法访问此页面" }}</p>
					<n-button type="primary" @click="refresh">
						<template #icon>
							<Icon name="mdi:refresh"></Icon>
						</template>
						重新获取审核信息
					</n-button>
				</div>
			</template>
		</n-result>
	</div>
</template>

<script lang="ts" setup>
import PublicToolbar from "@/layouts/common/Toolbar/PublicToolbar.vue"
import { useAuthStore } from "@/store/auth"

const store = useAuthStore()
const message = useMessage()
const router = useRouter()

async function refresh() {
	message.success("刷新成功")
}
</script>

<style lang="scss" scoped>
.layout {
	width: 100vw;
	height: 100vh;
}
</style>
