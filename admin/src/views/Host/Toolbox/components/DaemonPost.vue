<template>
  <n-drawer
    v-model:show="visible"
    :width="502"
    :mask-closable="false"
  >
    <n-drawer-content closable>
      <template #header>
        <div class="flex items-center gap-4">
          <n-button
            text
            @click="close"
          >
            <template #icon>
              <Icon name="mdi:arrow-left" />
            </template>
            {{$t("commons.button.back")}}
          </n-button>
          <n-divider vertical />
          <div>{{ title }}</div>
        </div>
      </template>
      <n-form
        ref="formRef"
        :model="formData"
        :rules="rules"
      >
        <n-form-item
          path="name"
          label="名称"
        >
          <n-input
            v-model:value="formData.name"
            :disabled="isUpdate"
          ></n-input>
        </n-form-item>
        <n-form-item
          path="directory"
          label="运行目录"
        >
          <n-input v-model:value="formData.directory">
            <template #prefix>
              <Icon name="clarity:directory-line" />
            </template>
          </n-input>
        </n-form-item>
        <n-form-item
          path="command"
          label="启动命令"
        >
          <n-input v-model:value="formData.command"></n-input>
        </n-form-item>
      </n-form>

      <n-collapse class="mt-4">
        <n-collapse-item
          title="其他设置"
          name="advanced"
        >
          <n-form-item
            path="user"
            label="启动用户"
          >
            <n-input v-model:value="formData.user"></n-input>
          </n-form-item>

          <n-form-item
            label="进程数量"
            path="numprocs"
          >
            <template #label>进程数量</template>
            <n-input-number
              v-model:value="formData.numprocs"
              clearable
            />
          </n-form-item>

          <n-form-item
            label="跟随Daemon启动"
            path="autostart"
          >
            <template #label>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <div style="display: flex; align-items: center; gap: 10">
                    跟随Daemon启动
                    <n-icon size="16">
                      <Icon name="mdi:help-circle-outline" />
                    </n-icon>
                  </div>
                </template>
                如果是true的话，子进程将在Daemond启动后被自动启动
              </n-tooltip>
            </template>
            <n-switch v-model:value="formData.autostart" />
          </n-form-item>
        </n-collapse-item>
      </n-collapse>

      <template #footer>
        <div class="flex justify-end gap-4">
          <n-button @click="close">{{$t("commons.button.cancel")}}</n-button>
          <n-button
            type="primary"
            @click="onConfirm"
            :loading="loading"
          >{{$t("commons.button.confirm")}}</n-button>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>
<script setup lang="ts">
import { ref } from "vue"
const visible = ref(false)
const loading = ref(false)
const isUpdate = ref(false)
const formData = ref({
	name: "",
	user: "root",
	directory: "",
	command: "",
	numprocs: 1,
	autostart: true
})
const rules = {
	name: {
		required: true,
		message: "请输入名称"
	},
	user: {
		required: true,
		message: "请输入启动用户"
	},
	directory: {
		required: true,
		message: "请输入启动目录"
	},
	command: {
		required: true,
		message: "请输入启动命令"
	},
	numprocs: {
		required: true,
		message: "请输入进程数量"
	}
}
const emit = defineEmits(["confirm"])
const formRef = ref()
const onConfirm = () => {
	formRef.value.validate((errors: any) => {
		if (errors) return
		emit("confirm", formData.value, loading)
	})
}

const title = ref("创建")
const open = (record?: any) => {
	visible.value = true
	if (record) {
		formData.value = {
			name: record.name,
			user: record.config.user,
			directory: record.config.directory,
			command: record.config.command,
			numprocs: record.config.numprocs,
			autostart: record.config.autostart,
			...record
		}
		isUpdate.value = true
		if (typeof record.config.user == "undefined") {
			formData.value.user = "root"
		}
		if (typeof record.config.autostart === "undefined") {
			formData.value.autostart = true
		}
		if (typeof record.config.numprocs === "undefined" || record.config.numprocs <= 0) {
			formData.value.numprocs = 1
		}
		title.value = "编辑"
	} else {
		title.value = "创建"
		formData.value = {
			name: "",
			user: "root",
			directory: "",
			command: "",
			numprocs: 1,
			autostart: true
		}
	}
}
const close = () => {
	visible.value = false
	loading.value = false
}
defineExpose({
	open,
	close
})
</script>
