<template>
	<n-drawer v-model:show="visible" :width="502" :mask-closable="false">
		<n-drawer-content closable>
			<template #header>
				<div class="flex items-center gap-4">
					<div class="flex items-center gap-2">
						<n-button text @click="close">
							<template #icon>
								<Icon name="oui:sort-left" />
							</template>
							返回
						</n-button>
					</div>
					<n-divider size="large" vertical />
					<div>初始化</div>
				</div>
			</template>
			<n-form :model="formData" ref="formRef" :rules="rules">
				<n-form-item path="path" label="主配置文件位置">
					<n-input v-model:value="formData.path" />
				</n-form-item>
				<n-form-item path="name" label="服务名称">
					<div>
						<n-input v-model:value="formData.name" />
						<div class="text-gray-400">systemctl 管理的 Daemon 服务名称，一般为 daemon、supervisord</div>
					</div>
				</n-form-item>
				<div class="rounded-lg bg-red-100 px-8 py-2 text-red-600">
					初始化操作需要修改配置文件的 [include] files
					参数，修改后的服务配置文件所在目录：panel安装目录/panel/tools/daemon/supervisor.d/
				</div>
				<div class="mt-8 rounded-lg bg-red-600 px-8 py-2 text-white">
					初始化会重启服务器，导致原有的守护进程全部关闭
				</div>
			</n-form>
			<template #footer>
				<div class="flex justify-end gap-2">
					<n-button @click="close">取消</n-button>
					<n-button type="primary" @click="confirm">确认</n-button>
				</div>
			</template>
		</n-drawer-content>
	</n-drawer>
</template>
<script setup lang="ts">
const visible = ref(false)
const formRef = ref()
const loading = ref(false)
const emit = defineEmits(["confirm"])
const formData = ref({
	path: "/etc/supervisord.conf",
	name: "supervisord"
})
const rules = {
	path: {
		required: true,
		message: "请输入主配置文件位置"
	},
	name: {
		required: true,
		message: "请输入服务名称"
	}
}
const open = () => {
	visible.value = true
}
const close = () => {
	visible.value = false
}
const confirm = () => {
	formRef.value.validate((errors: any) => {
		if (errors) return
		emit("confirm", formData.value, loading)
	})
}
defineExpose({
	open,
	close
})
</script>
