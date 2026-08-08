<template>
  <n-drawer
    v-model:show="drawerVisible"
    :width="600"
    placement="right"
  >
    <n-drawer-content
      title="{{ $t('container.createNetwork') }}"
      closable
    >
      <template #header>
        <div class="flex items-center">
          <div
            class="flex cursor-pointer items-center gap-2 text-gray-500"
            @click="handleClose"
          >
            <Icon name="mdi:arrow-left" />
            返回
          </div>
          <n-divider vertical />
          {{ $t('container.createNetwork') }}
        </div>
      </template>

      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-placement="left"
        label-width="100"
        class="py-4"
      >
        <n-form-item
          label="网络名"
          path="name"
        >
          <n-input
            v-model:value="form.name"
            placeholder="请输入网络名"
            clearable
          />
        </n-form-item>

        <n-form-item
          label="驱动"
          path="driver"
        >
          <n-select
            v-model:value="form.driver"
            :options="driverOptions"
            placeholder="请选择网络驱动"
          />
        </n-form-item>

        <n-form-item
          label="IPv4"
          path="ipv4"
        >
          <n-switch v-model:value="form.ipv4" />
        </n-form-item>

        <template v-if="form.ipv4">
          <n-form-item
            label="子网"
            path="subnet"
          >
            <n-input
              v-model:value="form.subnet"
              placeholder="例如: 172.16.10.0/24"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="网关"
            path="gateway"
          >
            <n-input
              v-model:value="form.gateway"
              placeholder="例如: 172.16.10.1"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="范围"
            path="scope"
          >
            <n-input
              v-model:value="form.scope"
              placeholder="例如: 172.16.10.0/16"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="辅助地址"
            path="auxAddress"
          >
            <n-space vertical>
              <n-space
                v-for="(address, idx) in form.auxAddress"
                :key="idx"
              >
                <n-input
                  v-model:value="address.key"
                  placeholder="Key"
                />
                <n-input
                  v-model:value="address.value"
                  placeholder="Value"
                />
                <n-button
                  @click="() => handleV4Delete(idx)"
                  type="error"
                >删除</n-button>
              </n-space>
              <n-button @click="handleV4Add">添加</n-button>
            </n-space>
          </n-form-item>
        </template>

        <n-form-item
          label="IPv6"
          path="ipv6"
        >
          <n-switch v-model:value="form.ipv6" />
        </n-form-item>

        <template v-if="form.ipv6">
          <n-form-item
            label="子网"
            path="subnetV6"
          >
            <n-input
              v-model:value="form.subnetV6"
              placeholder="例如: 2408:400e::/48"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="网关"
            path="gatewayV6"
          >
            <n-input
              v-model:value="form.gatewayV6"
              placeholder="例如: 2408:400e::1"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="范围"
            path="scopeV6"
          >
            <n-input
              v-model:value="form.scopeV6"
              placeholder="例如: 2408:400e::/64"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="辅助地址"
            path="auxAddressV6"
          >
            <n-space vertical>
              <n-space
                v-for="(address, idx) in form.auxAddressV6"
                :key="idx"
              >
                <n-input
                  v-model:value="address.key"
                  placeholder="Key"
                />
                <n-input
                  v-model:value="address.value"
                  placeholder="Value"
                />
                <n-button
                  @click="() => handleV6Delete(idx)"
                  type="error"
                >删除</n-button>
              </n-space>
              <n-button @click="handleV6Add">添加</n-button>
            </n-space>
          </n-form-item>
        </template>

        <n-form-item
          label="选项"
          path="options"
        >
          <n-input
            type="textarea"
            v-model:value="form.options"
            placeholder="一行一个选项"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </n-form-item>

        <n-form-item
          label="标签"
          path="labels"
        >
          <n-input
            type="textarea"
            v-model:value="form.labels"
            placeholder="一行一个标签"
            :autosize="{ minRows: 3, maxRows: 5 }"
          />
        </n-form-item>
      </n-form>

      <template #footer>
        <n-space>
          <n-button @click="handleClose">取消</n-button>
          <n-button
            type="primary"
            :loading="loading"
            @click="handleSubmit"
          >确认</n-button>
        </n-space>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue"
import { useMessage } from "naive-ui"
import type { FormInst, FormRules } from "naive-ui"
import { createNetwork } from "@/api/modules/container"
import type { Container } from "@/api/interface/container"

const props = defineProps<{
	show: boolean
}>()

const emit = defineEmits<{
	(e: "update:show", value: boolean): void
	(e: "success"): void
}>()

const message = useMessage()
const loading = ref(false)
const formRef = ref<FormInst | null>(null)

const drawerVisible = computed({
	get: () => props.show,
	set: value => emit("update:show", value)
})

const form = reactive({
	name: "",
	driver: "bridge",
	options: "",
	ipv4: true,
	subnet: "",
	gateway: "",
	scope: "",
	auxAddress: [] as Array<{ key: string; value: string }>,
	ipv6: false,
	subnetV6: "",
	gatewayV6: "",
	scopeV6: "",
	auxAddressV6: [] as Array<{ key: string; value: string }>,
	labels: ""
})

const driverOptions = [
	{ label: "bridge", value: "bridge" },
	{ label: "host", value: "host" },
	{ label: "none", value: "none" },
	{ label: "overlay", value: "overlay" },
	{ label: "macvlan", value: "macvlan" }
]

const rules: FormRules = {
	name: {
		required: true,
		message: "请输入网络名",
		trigger: "blur"
	},
	driver: {
		required: true,
		message: "请选择网络驱动",
		trigger: "change"
	},
	subnet: {
		required: true,
		message: "请输入子网",
		trigger: "blur"
	}
}

const handleClose = () => {
	drawerVisible.value = false
}

const handleV4Add = () => {
	form.auxAddress.push({
		key: "",
		value: ""
	})
}

const handleV4Delete = (index: number) => {
	form.auxAddress.splice(index, 1)
}

const handleV6Add = () => {
	form.auxAddressV6.push({
		key: "",
		value: ""
	})
}

const handleV6Delete = (index: number) => {
	form.auxAddressV6.splice(index, 1)
}

const handleSubmit = () => {
	formRef.value?.validate(async errors => {
		if (!errors) {
			try {
				loading.value = true
				const params: Container.NetworkCreate = {
					name: form.name,
					driver: form.driver,
					options: form.options ? form.options.split("\n").filter(Boolean) : [],
					labels: form.labels ? form.labels.split("\n").filter(Boolean) : [],
					subnet: form.subnet,
					gateway: form.gateway,
					scope: form.scope
				}

				await createNetwork(params)
				message.success("创建成功")
				handleClose()
				emit("success")
			} catch (error) {
				console.error("创建失败:", error)
			} finally {
				loading.value = false
			}
		}
	})
}
</script>

<style scoped>
.n-form-item {
	margin-bottom: 24px;
}
</style>
