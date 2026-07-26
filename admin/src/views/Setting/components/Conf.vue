<template>
  <div>
    <n-space
      vertical
      size="large"
      class="mt-4"
    >
      <div
        :bordered="false"
        size="small"
        class="bg-base-accent border-base-accent rounded-[28px] p-8"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
              Runtime Config
            </div>
            <div class="my-3 text-2xl font-semibold fg-base-100">常规设置</div>
          </div>
          <n-tag
            type="info"
            round
            :bordered="false"
          >运行参数</n-tag>
        </div>

        <n-form
          :model="form"
          label-width="120"
          class="mt-2"
        >
          <n-form-item label="运行端口">
            <n-input
              v-model:value="form.System.Port"
              class="mr-3"
              disabled
            />
            <n-button
              type="primary"
              @click="showPortDrawer = true"
            >设置</n-button>
          </n-form-item>

          <n-form-item
            label="安全入口"
            class="mt-2"
          >
            <n-input
              v-model:value="form.System.Entrance"
              class="mr-3"
              disabled
            />
            <n-button
              type="primary"
              @click="showEntranceDrawer = true"
            >设置</n-button>
          </n-form-item>

          <n-form-item label="基础目录">
            <n-input
              v-model:value="form.System.BaseDir"
              disabled
              class="field-input"
            />
          </n-form-item>

          <n-form-item label="日志目录">
            <n-input
              v-model:value="form.System.LogPath"
              disabled
              class="field-input"
            />
          </n-form-item>

          <n-form-item label="临时文件目录">
            <n-input
              v-model:value="form.System.TmpDir"
              disabled
              class="mr-3"
            />
            <n-button
              type="warning"
              class="ml-2"
              @click="clearTmp"
            >清空</n-button>
          </n-form-item>
        </n-form>
      </div>
      <div
        :bordered="false"
        size="small"
        class="bg-base-accent border-base-accent rounded-[28px] p-8 mt-6"
      >
        <div class="flex items-start justify-between gap-4">
          <div>
            <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">
              API Access
            </div>
            <div class="my-3 text-2xl font-semibold fg-base-100">API 接口配置</div>
          </div>
          <n-tag
            type="success"
            round
            :bordered="false"
          >开发者选项</n-tag>
        </div>

        <n-form
          :model="form"
          label-width="120"
          class="mt-2"
        >
          <n-form-item label="接口状态">
            <n-switch
              v-model:value="form.System.ApiInterfaceStatus"
              checked-value="Open"
              unchecked-value="Close"
            >
              <template #checked>已开启</template>
              <template #unchecked>已关闭</template>
            </n-switch>
          </n-form-item>

          <n-form-item label="API Token">
            <n-space>
              <n-input
                v-model:value="form.System.ApiKey"
                placeholder="请输入或生成随机 Token"
                class="mr-3 !w-[250px]"
              />

              <n-button
                type="primary"
                ghost
                class="mr-2"
                @click="generateApiToken"
              >随机生成</n-button>
              <n-button
                type="primary"
                :loading="loadingApiToken"
                @click="saveApiToken"
              >保存配置</n-button>
            </n-space>
          </n-form-item>
        </n-form>
      </div>
    </n-space>
    <n-modal
      v-model:show="showPortDrawer"
      preset="card"
      style="width: 400px;"
      placement="right"
    >
      <n-form label-width="120">
        <n-form-item label="绑定端口">
          <n-input
            v-model:value="editPort"
            class="drawer-input"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showPortDrawer = false">取消</n-button>
          <n-button
            type="primary"
            :loading="loadingPort"
            class="!rounded-[16px]"
            @click="confirmSetPort"
          >
            保存
          </n-button>
        </div>
      </template>
    </n-modal>

    <n-modal
      v-model:show="showEntranceDrawer"
      preset="card"
      style="width: 400px;"
      placement="right"
    >

      <n-form>
        <n-form-item label="新入口">
          <n-input
            v-model:value="editEntrance"
            class="drawer-input"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="showEntranceDrawer = false">取消</n-button>
          <n-button
            type="primary"
            :loading="loadingEntrance"
            class="!rounded-[16px]"
            @click="confirmSetEntrance"
          >
            保存
          </n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import {
	settingSystemClear,
	settingSystemConfig,
	settingSystemRestart,
	settingSystemEntrance,
	settingSystemPort,
	settingSystemApiTokenUpdate
} from "@/api/modules/setting"
import { computeSize } from "@/utils/util"
import { NButton, NCard, NDrawer, NDrawerContent, NForm, NFormItem, NInput, NSwitch, NTag, NSpace, useDialog, useMessage } from "naive-ui"
import { onMounted, reactive, ref } from "vue"

const message = useMessage()
const dialog = useDialog()
const form = reactive({
	System: {
		Port: "",
		Entrance: "",
		BaseDir: "",
		DbPath: "",
		LogPath: "",
		DataDir: "",
		TmpDir: "",
		Cache: "",
		Backup: "",
		ApiInterfaceStatus: "Close",
		ApiKey: ""
	},
	LogConfig: {}
})
const loadingPort = ref(false)
const loadingEntrance = ref(false)
const loadingApiToken = ref(false)
const showPortDrawer = ref(false)
const showEntranceDrawer = ref(false)
const editPort = ref("")
const editEntrance = ref("")

// 配置里的 system.port 形如 ":5470"，展示和提交都只用纯端口号
function normalizePort(raw: string): string {
	const value = String(raw ?? "").trim()
	if (!value) return ""
	const port = value.includes(":") ? value.slice(value.lastIndexOf(":") + 1) : value
	return port.trim()
}

async function fetchConfig() {
	try {
		const res = await settingSystemConfig()
		if (res.data && res.data.System) {
			Object.assign(form.System, res.data.System)
			form.System.Port = normalizePort(res.data.System.Port)
			editPort.value = form.System.Port
			editEntrance.value = res.data.System.Entrance || ""
			if (res.data.LogConfig) form.LogConfig = res.data.LogConfig
		}
	} catch {
		message.error("获取配置失败")
	}
}

function jumpToNewUrl(port: string, entrance: string) {
	let protocol = window.location.protocol
	let host = window.location.hostname
	let newUrl = `${protocol}//${host}:${normalizePort(port)}/${entrance}`
	window.location.href = newUrl
}

function confirmSetPort() {
	const port = Number(normalizePort(editPort.value))
	if (!Number.isInteger(port) || port < 1 || port > 65535) {
		message.warning("端口需为 1-65535 之间的整数")
		return
	}
	if (port === Number(form.System.Port)) {
		message.info("端口未发生变化")
		return
	}
	dialog.warning({
		title: "提示",
		content: `修改后面板将重启并监听 ${port} 端口，是否继续？`,
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			loadingPort.value = true
			try {
				// 端口生效的重启由服务端自己安排（见 SettingSystemPort），
				// 前端不能再调 restart 后立刻跳转 —— 那个请求会被跳转取消，导致改了配置却没重启
				await settingSystemPort({ serverPort: port })
				message.success("端口设置成功，面板正在重启，稍后将跳转到新地址")
				showPortDrawer.value = false
				form.System.Port = String(port)
				setTimeout(() => jumpToNewUrl(String(port), form.System.Entrance), 4000)
			} catch {
				// 失败信息由请求拦截器统一提示
			}
			loadingPort.value = false
		}
	})
}

function clearAllCookies() {
	document.cookie.split(";").forEach(c => {
		document.cookie = c.replace(/^ +/, "").replace(/=.*/, `=;expires=${new Date(0).toUTCString()};path=/`)
	})
}

function confirmSetEntrance() {
	dialog.warning({
		title: "提示",
		content: "修改后系统将自动重启，是否继续？",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			if (!editEntrance.value || !/^[\w-]+$/.test(editEntrance.value)) {
				message.warning("安全入口只能包含字母、数字、下划线和短横线")
				return
			}
			loadingEntrance.value = true
			try {
				await settingSystemEntrance({ entrance: editEntrance.value })
				message.success("安全入口设置成功，系统即将重启")
				showEntranceDrawer.value = false
				clearAllCookies()
				form.System.Entrance = editEntrance.value
				await settingSystemRestart()
				setTimeout(() => jumpToNewUrl(form.System.Port, editEntrance.value), 4000)
			} catch {
				// message.error("安全入口设置失败")
			}
			loadingEntrance.value = false
		}
	})
}

function clearLog() {
	dialog.warning({
		title: "确认清空",
		content: "确定要清空日志目录吗？此操作不可恢复。",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const res = await settingSystemClear({ key: "log" })
				if (res.data) {
					message.success(`清理了${res.data.count}个文件，释放了${computeSize(res.data.total_size)}空间`)
				} else {
					message.success("日志目录清空成功")
				}
			} catch {
			}
		}
	})
}

function clearTmp() {
	dialog.warning({
		title: "确认清空",
		content: "确定要清空临时目录吗？此操作不可恢复。",
		positiveText: "确定",
		negativeText: "取消",
		onPositiveClick: async () => {
			try {
				const res = await settingSystemClear({ key: "tmp" })
				if (res.data) {
					message.success(`清理了${res.data.count}个文件，释放了${computeSize(res.data.total_size)}空间`)
				} else {
					message.success("临时目录清空成功")
				}
			} catch {
			 
			}
		}
	})
}

function generateApiToken() {
	const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
	let token = ''
	for (let i = 0; i < 32; i++) {
		token += chars.charAt(Math.floor(Math.random() * chars.length))
	}
	form.System.ApiKey = token
}

async function saveApiToken() {
	if (form.System.ApiInterfaceStatus === 'Open' && !form.System.ApiKey) {
		message.warning("开启 API 接口时，必须配置 API Token")
		return
	}
	loadingApiToken.value = true
	try {
		await settingSystemApiTokenUpdate({
			apiInterfaceStatus: form.System.ApiInterfaceStatus,
			apiKey: form.System.ApiKey
		})
		message.success("API 配置保存成功")
	} catch {
		message.error("API 配置保存失败")
	} finally {
		loadingApiToken.value = false
	}
}

onMounted(() => {
	fetchConfig()
})
</script>
