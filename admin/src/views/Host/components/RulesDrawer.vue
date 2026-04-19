<template>
  <n-drawer
    :show="visible"
    :width="720"
    placement="right"
    :mask-closable="false"
    :closable="true"
    @update:show="handleShowChange"
    @close="close"
  >
    <n-drawer-content>
      <template #header>
        <div class="flex items-center">
          <div
            class="flex cursor-pointer items-center gap-2 text-slate-500"
            @click="close"
          >
            <Icon name="mdi:arrow-left" />
            返回
          </div>
          <n-divider vertical />
          {{ title }}
        </div>
      </template>
      <n-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-placement="top"
      >
        <n-form-item
          v-if="currentType !== 'ip'"
          label="协议"
          path="protocol"
        >
          <n-select
            :value="formData.protocol"
            :options="protocolOptions"
            @update:value="handleProtocolChange"
          />
        </n-form-item>

        <n-form-item
          v-if="currentType === 'port' || currentType === 'forward'"
          :label="currentType === 'forward' ? '入口端口' : '端口'"
          path="port"
        >
          <div class="flex-1">
            <n-input
              v-model:value="formData.port"
              placeholder="请输入端口号"
              @update:value="handlePortChange"
            />
            <div class="mt-4 text-sm text-slate-500">
              <div>多个端口支持逗号分隔，例如：8080,8081</div>
              <div class="mt-2">范围端口支持连字符，例如：8080-8089</div>
            </div>
          </div>
        </n-form-item>

        <n-form-item
          v-if="currentType === 'ip'"
          label="指定 IP"
          path="address"
        >
          <div class="flex-1">
            <n-input
              v-model:value="formData.address"
              placeholder="请输入指定 IP 或 IP 段"
              @update:value="handleAddressChange"
            />
            <div class="mt-4 text-sm text-slate-500">
              <div>支持输入 IP 或网段，例如：172.16.10.11 或 172.16.0.0/24</div>
              <div class="mt-2">多个 IP 或网段请使用英文逗号分隔</div>
            </div>
          </div>
        </n-form-item>

        <n-grid
          v-if="currentType === 'forward'"
          :cols="2"
          :x-gap="16"
        >
          <n-form-item-gi
            label="目标 IP"
            path="targetIP"
          >
            <n-input
              v-model:value="formData.targetIP"
              placeholder="默认 127.0.0.1"
              @update:value="handleTargetIPChange"
            />
          </n-form-item-gi>
          <n-form-item-gi
            label="目标端口"
            path="targetPort"
          >
            <n-input
              v-model:value="formData.targetPort"
              placeholder="请输入目标端口"
              @update:value="handleTargetPortChange"
            />
          </n-form-item-gi>
        </n-grid>

        <n-form-item
          v-if="currentType !== 'forward'"
          label="策略"
          path="strategy"
        >
          <n-radio-group
            :value="formData.strategy"
            @update:value="handleStrategyChange"
          >
            <n-radio value="accept">允许</n-radio>
            <n-radio value="drop">拒绝</n-radio>
          </n-radio-group>
        </n-form-item>

        <n-form-item
          v-if="currentType !== 'forward'"
          label="描述信息"
          path="description"
        >
          <n-input
            :value="formData.description"
            placeholder="请输入规则描述信息"
            @update:value="handleDescriptionChange"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex items-center justify-end gap-4">
          <n-button @click="close">取消</n-button>
          <n-button
            type="primary"
            :loading="loading"
            @click="save"
          >保存</n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, ref } from "vue"

const emit = defineEmits(["save"])

const protocolOptions = [
	{ label: "tcp", value: "tcp" },
	{ label: "udp", value: "udp" },
	{ label: "tcp/udp", value: "tcp/udp" }
]

const visible = ref(false)
const loading = ref(false)
const currentType = ref<"port" | "ip" | "forward">("port")
const editMode = ref(false)
const originalData = ref<any>(null)
const title = computed(() => {
	const actionText = editMode.value ? "编辑" : "创建"
	if (currentType.value === "ip") return `${actionText} IP 规则`
	if (currentType.value === "forward") return `${actionText}转发规则`
	return `${actionText}端口规则`
})

const formData = ref<any>({})
const formRef = ref()

const resetForm = () => {
	formData.value = {
		protocol: "tcp",
		port: "",
		strategy: "accept",
		operation: "add",
		address: "",
		description: "",
		targetIP: "127.0.0.1",
		targetPort: "",
		num: ""
	}
}

resetForm()

const formRules = computed(() => {
	const rules: Record<string, any> = {}
	if (currentType.value !== "ip") {
		rules.protocol = { required: true, message: "请选择协议类型", trigger: "blur" }
	}
	if (currentType.value === "port" || currentType.value === "forward") {
		rules.port = { required: true, message: "请输入端口号", trigger: "blur" }
	}
	if (currentType.value === "ip") {
		rules.address = { required: true, message: "请输入指定 IP", trigger: "blur" }
	}
	if (currentType.value === "forward") {
		rules.targetPort = { required: true, message: "请输入目标端口", trigger: "blur" }
	}
	return rules
})

const open = (type: "port" | "ip" | "forward", data?: any) => {
	currentType.value = type
	editMode.value = Boolean(data)
	originalData.value = data ? { ...data } : null
	resetForm()
	if (data) {
		formData.value = {
			...formData.value,
			...data,
			strategy: data.strategy || "accept",
			protocol: data.protocol || "tcp",
			targetIP: data.targetIP || "127.0.0.1",
			operation: data.operation || "add"
		}
	}
	visible.value = true
}

const close = () => {
	visible.value = false
}

const handleShowChange = (value: boolean) => {
	visible.value = value
}

const handleProtocolChange = (value: string) => {
	formData.value.protocol = value
}

const handlePortChange = (value: string) => {
	formData.value.port = value
}

const handleAddressChange = (value: string) => {
	formData.value.address = value
}

const handleTargetIPChange = (value: string) => {
	formData.value.targetIP = value
}

const handleTargetPortChange = (value: string) => {
	formData.value.targetPort = value
}

const handleStrategyChange = (value: string) => {
	formData.value.strategy = value
}

const handleDescriptionChange = (value: string) => {
	formData.value.description = value
}

const save = () => {
	formRef.value?.validate((errors: any) => {
		if (errors) return
		emit(
			"save",
			{
				type: currentType.value,
				isEdit: editMode.value,
				data: { ...formData.value },
				oldData: originalData.value ? { ...originalData.value } : null
			},
			loading
		)
	})
}

defineExpose({
	open,
	close
})
</script>
