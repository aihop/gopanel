<template>
  <div class="mobile-auth-container flex h-screen w-full items-center justify-center bg-slate-50 px-4">
    <div class="auth-card w-full max-w-sm rounded-[24px] bg-base-100 p-8 shadow-[0_10px_40px_rgba(15,23,42,0.08)]">
      <div class="mb-8 flex flex-col items-center justify-center">
        <Logo
          :mini="true"
          class="mb-4"
        />
        <h2 class="text-xl font-bold fg-base-100">授权登录面板</h2>
        <p class="mt-2 text-center text-sm text-slate-500">您正在请求通过移动设备授权登录关联服务器</p>
      </div>

      <div class="space-y-4">
        <div v-if="authStatus === 'idle'">
          <n-button
            type="primary"
            block
            size="large"
            @click="handleAuth"
            :loading="loading"
          >
            确认授权登录
          </n-button>
          <n-button
            block
            size="large"
            class="mt-6"
            @click="handleCancel"
          >取消</n-button>
        </div>

        <div
          v-else-if="authStatus === 'success'"
          class="flex flex-col items-center justify-center py-4"
        >
          <Icon
            :size="48"
            class="mb-4"
            color="#22c55e"
            name="carbon:checkmark-filled"
          />
          <h3 class="text-lg font-semibold fg-base-100">授权成功</h3>
          <p class="mt-2  mb-6 text-center text-sm text-slate-500">您已成功授权，网页端将自动登录</p>
          <n-button
            block
            size="large"
            @click="handleClose"
          >关闭页面</n-button>
        </div>

        <div
          v-else-if="authStatus === 'error'"
          class="flex flex-col items-center justify-center py-4"
        >
          <Icon
            :size="48"
            class="mb-4"
            color="#ef4444"
            name="carbon:warning-filled"
          />
          <h3 class="text-lg font-semibold fg-base-100">授权失败</h3>
          <p class="mt-2 text-center text-sm text-slate-500">
            {{ errorMessage || "授权过程中发生错误，请重试" }}
          </p>
          <n-button
            type="primary"
            block
            size="large"
            class="mt-6"
            @click="retryAuth"
          >重新授权</n-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from "vue"
import { useRoute } from "vue-router"
import Logo from "@/layouts/common/Logo.vue"
import Icon from "@/components/common/Icon.vue"
import { Icon as Iconify } from "@iconify/vue"
import { NButton } from "naive-ui"

const route = useRoute()
const token = ref<string>("")
const loading = ref(false)
const authStatus = ref<"idle" | "success" | "error">("idle")
const errorMessage = ref("")

onMounted(() => {
	// 获取 URL 中的 token
	if (route.query.token) {
		token.value = route.query.token as string
	} else {
		authStatus.value = "error"
		errorMessage.value = "缺少授权令牌 (Token)"
	}
})

const handleAuth = async () => {
	if (!token.value) return

	loading.value = true
	try {
		// 这里模拟请求后端接口，将 token 和当前手机登录状态进行绑定
		await new Promise(resolve => setTimeout(resolve, 1500))

		// 假设授权成功
		authStatus.value = "success"
	} catch (error: any) {
		authStatus.value = "error"
		errorMessage.value = error.message || "网络请求失败"
	} finally {
		loading.value = false
	}
}

const handleCancel = () => {
	// 可以根据实际情况返回首页或者关闭窗口
	authStatus.value = "error"
	errorMessage.value = "您已取消授权"
}

const retryAuth = () => {
	authStatus.value = "idle"
	errorMessage.value = ""
}

const handleClose = () => {
	// 尝试关闭页面或返回应用首页
	if (window.history.length > 1) {
		window.history.back()
	} else {
		window.close()
	}
}
</script>

<style scoped lang="scss">
.mobile-auth-container {
	min-height: 100vh;
	min-height: 100dvh; // 适配移动端
}
</style>
