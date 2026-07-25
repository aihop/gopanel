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

          <n-form-item
            label="自动重启"
            path="autorestart"
          >
            <template #label>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <div style="display: flex; align-items: center; gap: 10">
                    自动重启
                    <n-icon size="16">
                      <Icon name="mdi:help-circle-outline" />
                    </n-icon>
                  </div>
                </template>
                异常退出：仅当退出码非 0（程序异常终止）时重启；总是：任何退出都重启；从不：不重启
              </n-tooltip>
            </template>
            <n-select
              v-model:value="formData.autorestart"
              :options="autorestartOptions"
              style="width: 220px"
            />
          </n-form-item>

          <n-form-item
            label="启动重试次数"
            path="startretries"
          >
            <template #label>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <div style="display: flex; align-items: center; gap: 10">
                    启动重试次数
                    <n-icon size="16">
                      <Icon name="mdi:help-circle-outline" />
                    </n-icon>
                  </div>
                </template>
                连续启动失败达到该次数后放弃（FATAL 状态）。进程成功运行超过“启动稳定时长”后，该计数会清零
              </n-tooltip>
            </template>
            <n-input-number
              v-model:value="formData.startretries"
              :min="1"
              :max="100"
            />
          </n-form-item>

          <n-form-item
            label="启动稳定时长(秒)"
            path="startsecs"
          >
            <template #label>
              <n-tooltip trigger="hover">
                <template #trigger>
                  <div style="display: flex; align-items: center; gap: 10">
                    启动稳定时长(秒)
                    <n-icon size="16">
                      <Icon name="mdi:help-circle-outline" />
                    </n-icon>
                  </div>
                </template>
                进程需持续运行超过该秒数才算“启动成功”并清零重试计数，默认 1
              </n-tooltip>
            </template>
            <n-input-number
              v-model:value="formData.startsecs"
              :min="1"
              :max="300"
            />
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
const originalConfig = ref<Record<string, any> | null>(null)
const autorestartOptions = [
	{ label: "异常退出时重启 (unexpected)", value: "unexpected" },
	{ label: "总是重启 (always)", value: "true" },
	{ label: "从不重启 (never)", value: "false" }
]
const formData = ref<Record<string, any>>({
	name: "",
	user: "root",
	directory: "",
	command: "",
	numprocs: 1,
	autostart: true,
	autorestart: "unexpected",
	startretries: 3,
	startsecs: 1
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
const buildPayload = () => ({
        ...(originalConfig.value || {}),
        ...formData.value
})
const onConfirm = () => {
	formRef.value.validate((errors: any) => {
		if (errors) return
                emit("confirm", { data: buildPayload(), isUpdate: isUpdate.value }, loading)
	})
}

const title = ref("创建")
const open = (record?: any) => {
	visible.value = true
	if (record) {
                const config = record.config || {}
                originalConfig.value = { ...config }
		formData.value = {
			name: record.name,
                        user: config.user,
                        directory: config.directory,
                        command: config.command,
                        numprocs: config.numprocs,
                        autostart: config.autostart,
                        autorestart: config.autorestart,
                        startretries: config.startretries,
                        startsecs: config.startsecs
		}
		isUpdate.value = true
                if (typeof config.user == "undefined") {
			formData.value.user = "root"
		}
                if (typeof config.autostart === "undefined") {
			formData.value.autostart = true
		}
                if (typeof config.numprocs === "undefined" || config.numprocs <= 0) {
			formData.value.numprocs = 1
		}
                if (typeof config.autorestart === "undefined" || config.autorestart === "") {
			formData.value.autorestart = "unexpected"
		}
                if (typeof config.startretries === "undefined") {
			formData.value.startretries = 3
		}
                if (typeof config.startsecs === "undefined") {
			formData.value.startsecs = 1
		}
		title.value = "编辑"
	} else {
                isUpdate.value = false
                originalConfig.value = null
		title.value = "创建"
		formData.value = {
			name: "",
			user: "root",
			directory: "",
			command: "",
			numprocs: 1,
			autostart: true,
			autorestart: "unexpected",
			startretries: 3,
			startsecs: 1
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
