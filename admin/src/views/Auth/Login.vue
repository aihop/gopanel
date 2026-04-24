<template>
  <div
    v-if="!isLogged"
    :class="isDark ? 'bg-slate-950 text-slate-100' : 'bg-slate-50 fg-base-100'"
  >
    <div class="pointer-events-none absolute inset-0 overflow-hidden">
      <div
        class="absolute -left-16 top-0 h-72 w-72 rounded-full blur-3xl"
        :class="isDark ? 'bg-blue-500/18' : 'bg-blue-300/35'"
      ></div>
      <div
        class="absolute right-[-80px] top-[12%] h-96 w-96 rounded-full blur-3xl"
        :class="isDark ? 'bg-cyan-400/12' : 'bg-sky-200/55'"
      ></div>
      <div
        class="absolute bottom-[-120px] left-1/3 h-80 w-80 rounded-full blur-3xl"
        :class="isDark ? 'bg-indigo-500/14' : 'bg-indigo-200/50'"
      ></div>
      <div
        class="absolute inset-0 opacity-[0.55]"
        :class="
					isDark
						? 'bg-[radial-gradient(circle_at_top,_rgba(30,41,59,0.55),_transparent_42%),linear-gradient(135deg,rgba(2,6,23,0.94),rgba(15,23,42,0.98))]'
						: 'bg-[radial-gradient(circle_at_top,_rgba(191,219,254,0.55),_transparent_42%),linear-gradient(135deg,rgba(239,246,255,0.96),rgba(248,250,252,0.98))]'
				"
      ></div>
    </div>

    <div class="relative z-10 flex min-h-screen items-center justify-center px-4 py-10 sm:px-6 lg:px-8">
      <div
        class="w-full max-w-[1180px] overflow-hidden rounded-[36px] border shadow-[0_30px_80px_rgba(15,23,42,0.18)] backdrop-blur-2xl"
        :class="isDark ? 'bg-slate-900/78 border-slate-800' : 'bg-white/78 border-blue-100/80'"
      >
        <div class="grid min-h-[760px] grid-cols-1 lg:grid-cols-[1.05fr_0.95fr]">
          <div
            class="flex flex-col justify-between p-8 sm:p-10 lg:p-14  hidden md:block"
            :class="isDark ? 'bg-slate-900/56' : 'bg-transparent'"
          >
            <div class="space-y-8">
              <div class="flex items-center justify-between gap-4">
                <Logo
                  :mini="false"
                  :dark="isDark"
                  max-height="46px"
                />
                <div
                  class="inline-flex h-10 items-center rounded-full border px-4 text-xs font-bold uppercase tracking-[0.18em]"
                  :class="
										isDark
											? 'border-blue-400/20 bg-blue-400/10 text-blue-200'
											: 'border-blue-200 bg-white/70 text-blue-600'
									"
                >
                  {{ appBrand }}
                </div>
              </div>

              <div class="space-y-5">
                <div
                  class="text-xs font-bold uppercase tracking-[0.22em]"
                  :class="isDark ? 'text-blue-200/85' : 'text-blue-600'"
                >
                  {{ appBrand }}
                </div>
                <h1 class="font-family-[PingFang SC] max-w-[620px] text-4xl font-bold leading-[1.05] sm:text-5xl lg:text-[56px]">
                  安全、好用的服务器面板
                </h1>
                <p
                  class="max-w-[620px] text-base leading-8 sm:text-lg"
                  :class="isDark ? 'text-slate-300/88' : 'text-slate-500'"
                >
                  {{ appBrand }} 围绕网站、数据库、Docker、应用商店与系统运维进行一体化设计，
                  用更克制的视觉、更清晰的结构和更顺手的交互，把复杂的服务器管理变成真正好看、好用、好维护的日常工作台。
                </p>
              </div>

              <div class="grid gap-4 sm:grid-cols-3">
                <div
                  v-for="item in heroHighlights"
                  :key="item.label"
                  class="rounded-[24px] border p-5"
                  :class="
										isDark ? 'border-slate-700/70 bg-slate-800/60' : 'bg-white/72 border-blue-100'
									"
                >
                  <div
                    class="text-xs font-bold uppercase tracking-[0.18em]"
                    :class="isDark ? 'text-blue-200/85' : 'text-blue-600'"
                  >
                    {{ item.label }}
                  </div>
                  <div class="mt-3 text-xl font-semibold">{{ item.value }}</div>
                  <div
                    class="mt-2 text-sm leading-6"
                    :class="isDark ? 'text-slate-400' : 'text-slate-500'"
                  >
                    {{ item.desc }}
                  </div>
                </div>
              </div>
            </div>

            <div
              class="mt-10 grid gap-3 sm:grid-cols-3"
              :class="isDark ? 'text-slate-300' : 'text-slate-600'"
            >
              <div
                v-for="tag in heroTags"
                :key="tag"
                class="rounded-2xl border px-4 py-3 text-sm font-medium"
                :class="isDark ? 'border-slate-700/80 bg-slate-800/50' : 'bg-white/72 border-blue-100'"
              >
                {{ tag }}
              </div>
            </div>
          </div>

          <div
            class="flex items-center justify-center p-5 sm:p-8 lg:p-10"
            :class="isDark ? 'bg-slate-950/34' : 'bg-white/20'"
          >
            <div
              class="w-full max-w-[460px] rounded-[28px] border p-7 shadow-[0_24px_60px_rgba(15,23,42,0.14)] sm:p-9"
              :class="isDark ? 'bg-slate-900/88 border-slate-700/80' : 'bg-white/92 border-white/80'"
            >
              <div class="mb-8">
                <div
                  class="text-xs font-bold uppercase tracking-[0.2em]"
                  :class="isDark ? 'text-blue-200/85' : 'text-blue-600'"
                >
                  Secure Access
                </div>
                <div class="mt-3 text-3xl font-bold">{{ $t("auth.signin") }}</div>
                <div
                  class="mt-3 text-sm leading-7"
                  :class="isDark ? 'text-slate-400' : 'text-slate-500'"
                >
                  使用你的 {{ appBrand }} 管理员账号登录，进入统一的服务器控制台。
                </div>
              </div>

              <n-form
                ref="formRef"
                :model="model"
                :rules="rules"
                label-placement="top"
              >
                <n-form-item
                  path="email"
                  :label="$t('auth.email')"
                  first
                >
                  <n-input
                    :value="model.email"
                    size="large"
                    placeholder="admin@example.com"
                    autocomplete="on"
                    :style="inputStyle"
                    @update:value="value => (model.email = value)"
                    @keydown.enter="signIn"
                  />
                </n-form-item>
                <n-form-item
                  path="password"
                  :label="$t('auth.password')"
                >
                  <n-input
                    :value="model.password"
                    type="password"
                    show-password-on="click"
                    autocomplete="on"
                    size="large"
                    :placeholder="$t('auth.password')"
                    :style="inputStyle"
                    @update:value="value => (model.password = value)"
                    @keydown.enter="signIn"
                  />
                </n-form-item>

                <div
                  v-if="captchaRequired"
                  class="mb-5 rounded-2xl border px-4 py-3 text-sm leading-6"
                  :class="isDark ? 'border-amber-400/20 bg-amber-400/10 text-amber-100' : 'border-amber-200 bg-amber-50 text-amber-700'"
                >
                  检测到一次登录失败，后续登录前需要先完成滑块验证。
                </div>

                <div class="mt-2 flex flex-col gap-6">
                  <div class="flex items-center justify-between gap-4">
                    <n-checkbox
                      :checked="rememberMe"
                      size="large"
                      @update:checked="value => (rememberMe = value)"
                    >
                      {{ $t("auth.remember_me") }}
                    </n-checkbox>

                  </div>

                  <n-button
                    type="primary"
                    size="large"
                    class="!h-14 !rounded-[20px] !text-base !font-bold shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
                    :disabled="!isValid"
                    :style="buttonStyle"
                    @click="signIn"
                  >
                    {{ $t("auth.signin") }}
                  </n-button>
                </div>
              </n-form>

              <LoginCaptcha
                ref="captchaRef"
                @success="handleCaptchaSuccess"
              />
            </div>
          </div>
        </div>
        <div class="text-center text-gray-500 p-3 text-sm font-medium">
          @Copyright 2026
          <a
            href="https://gopanel.cn/"
            target="_blank"
          >
            GoPanel
          </a>
          All Rights Reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { FormType } from "@/components/auth/types.d"
import type { FormInst, FormRules, FormValidationError } from "naive-ui"
import { authSignInAPI } from "@/api/modules/auth"
import LoginCaptcha from "@/components/auth/LoginCaptcha.vue"
import Logo from "@/layouts/common/Logo.vue"
import { useAuthStore } from "@/store/auth"
import GlobalStore from "@/store/modules/global"
import { useThemeStore } from "@/store/theme"
import { useMessage, useNotification } from "naive-ui"
import { computed, onBeforeMount, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import { useRoute, useRouter } from "vue-router"

const { formType } = defineProps<{
	formType?: FormType
}>()

interface ModelType {
	email: string | null
	password: string | null
}

interface LoginCaptchaExpose {
	show: () => void
	close: () => void
}

const { t } = useI18n()
const themeStore = useThemeStore()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const globalStore = GlobalStore()
const message = useMessage()
const notification = useNotification()

const isDark = computed(() => themeStore.isThemeDark)
const type = ref<FormType | undefined>(formType || undefined)
const isLogged = computed(() => authStore.isLogged)
const appBrand = import.meta.env.VITE_APP_BRAND || "GoPanel"
const rememberMe = ref(localStorage.getItem("gopanel-admin-remembered") === "true")
const formRef = ref<FormInst | null>(null)
const captchaRef = ref<LoginCaptchaExpose | null>(null)
const captchaRequired = ref(false)
const captchaToken = ref("")
const pendingSubmitAfterCaptcha = ref(false)

const model = ref<ModelType>({
	email: localStorage.getItem("gopanel-admin-email") || "",
	password: localStorage.getItem("gopanel-admin-password") || ""
})

// 如果当前访问的域名是 demo.gopanel.cn，直接跳转到演示页面
// 就将账户密码自动天输入表单
onBeforeMount(() => {
	if (window.location.hostname === "demo.gopanel.run") {
		model.value.email = "demo@gopanel.run"
		model.value.password = "123456"
	}
	if (typeof route.query.entrance === "string" && route.query.entrance.trim()) {
		globalStore.setEntrance(route.query.entrance.trim())
	}
})

const heroHighlights = computed(() => [
	{
		label: "Website",
		value: "网站与代理",
		desc: "统一管理站点、反向代理、证书与运行环境。"
	},
	{
		label: "Docker",
		value: "容器与编排",
		desc: "覆盖容器、镜像、网络、卷与 Compose 日常运维。"
	},
	{
		label: "Panel",
		value: "简约且实用",
		desc: "延续 Dashboard 的蓝色、扁平、低噪声设计与高效率交互。"
	}
])

const heroTags = computed(() => ["网站管理", "数据库与备份", "Docker / Compose"])

const rules: FormRules = {
	email: [
		{
			required: true,
			trigger: ["blur"],
			message: t("form.requiredTo", [t("auth.email")])
		}
	],
	password: [
		{
			required: true,
			trigger: ["blur"],
			message: t("form.requiredTo", [t("auth.password")])
		}
	]
}

const isValid = computed(() => Boolean(model.value.email && model.value.password))

const inputStyle = computed(() => ({
	"--n-height": "60px",
	"--n-border-radius": "20px",
	"--n-padding-left": "20px",
	"--n-padding-right": "20px",
	"--n-font-size": "16px",
	"--n-color": isDark.value ? "rgba(15,23,42,0.86)" : "rgba(248,250,252,0.96)",
	"--n-color-focus": isDark.value ? "rgba(15,23,42,0.96)" : "rgba(255,255,255,1)",
	"--n-color-disabled": isDark.value ? "rgba(30,41,59,0.85)" : "rgba(241,245,249,0.92)",
	"--n-border": isDark.value ? "1px solid rgba(71,85,105,0.95)" : "1px solid rgba(203,213,225,0.95)",
	"--n-border-hover": "1px solid rgba(147,197,253,0.95)",
	"--n-border-focus": "1px solid rgba(37,99,235,0.95)",
	"--n-box-shadow-focus": "0 0 0 4px rgba(37,99,235,0.12)"
}))

const buttonStyle = computed(() => ({
	"--n-border-radius": "20px",
	"--n-height": "56px"
}))

async function signIn() {
	formRef.value?.validate(async (errors: Array<FormValidationError> | undefined) => {
		if (errors) {
			message.error("Invalid credentials")
			return
		}

		if (captchaRequired.value && !captchaToken.value) {
			pendingSubmitAfterCaptcha.value = true
			captchaRef.value?.show()
			return
		}

		try {
			const { data } = await authSignInAPI({
				email: model.value.email!,
				password: model.value.password!,
				captchaToken: captchaToken.value || undefined
			})

			if (rememberMe.value) {
				localStorage.setItem("gopanel-admin-email", model.value.email!)
				localStorage.setItem("gopanel-admin-password", model.value.password!)
				localStorage.setItem("gopanel-admin-remembered", "true")
			} else {
				localStorage.removeItem("gopanel-admin-email")
				localStorage.removeItem("gopanel-admin-password")
				localStorage.removeItem("gopanel-admin-remembered")
			}

			authStore.setLogged({
				auth: data.xAuth,
				user: data.userInfo
			})
			captchaRequired.value = false
			captchaToken.value = ""
			pendingSubmitAfterCaptcha.value = false

			await router.push({ path: "/", replace: true })
			notification.success({
				title: t("auth.login_success"),
				content: t("auth.login_success_tips"),
				duration: 3000
			})
		} catch (error) {
			console.error(error)
			captchaRequired.value = true
			captchaToken.value = ""
			pendingSubmitAfterCaptcha.value = false
		}
	})
}

function handleCaptchaSuccess(token: string) {
	captchaToken.value = token
	if (pendingSubmitAfterCaptcha.value) {
		pendingSubmitAfterCaptcha.value = false
		void signIn()
	}
}

onBeforeMount(() => {
	if (route.query.step) {
		type.value = route.query.step as FormType
	}
})

onMounted(() => {
	if (isLogged.value) {
		router.push({ path: "/", replace: true })
	}
})

watch(isValid, val => {
	if (val) {
		formRef.value?.validate()
	}
})
</script>
