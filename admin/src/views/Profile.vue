<template>
  <div class="relative min-h-full overflow-hidden rounded-[28px] bg-slate-50 px-4 py-6 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-[520px] rounded-[30px] border border-slate-200 bg-base-100 p-7 shadow-[0_18px_48px_rgba(15,23,42,0.06)] sm:p-9">
      <div class="flex items-start justify-between gap-4">
        <div>
          <div class="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">Password Settings</div>
          <div class="mt-3 text-3xl font-semibold fg-base-100">{{ $t('user.changeLoginPassword') }}</div>
          <div class="mt-3 flex items-center gap-3 text-sm leading-7 text-slate-500">
            <div>
              {{ $t('user.currentAccount') }}：<span class="font-medium text-slate-700">{{ authStore.nickName }}</span>
              <span
                v-if="user?.email"
                class="ml-1 text-slate-400"
              >· {{ user.email }}</span>
            </div>
            <n-button
              size="tiny"
              quaternary
              type="primary"
              @click="showEditInfoModal = true"
            >
              修改资料
            </n-button>
          </div>
        </div>
        <div class="rounded-2xl border border-blue-100 bg-blue-50 px-4 py-3 text-right">
          <div class="text-xs font-semibold uppercase tracking-[0.14em] text-blue-600">{{ $t('user.safeSuggestions') }}</div>
          <div class="mt-2 text-sm text-slate-600">{{ $t('user.passwordSuggestions_tips') }}</div>
        </div>
      </div>

      <n-form
        class="mt-8"
        label-placement="top"
      >
        <n-form-item :label="$t('user.oldPassword')">
          <n-input
            :value="password"
            type="password"
            size="large"
            show-password-on="click"
            :style="inputStyle"
            @update:value="handlePasswordChange"
          />
        </n-form-item>
        <n-form-item :label="$t('user.newPassword')">
          <n-input
            :value="newPassword"
            type="password"
            size="large"
            show-password-on="click"
            :style="inputStyle"
            @update:value="handleNewPasswordChange"
          />
        </n-form-item>
        <n-form-item :label="$t('user.confirmPassword')">
          <n-input
            :value="confirmPassword"
            type="password"
            size="large"
            show-password-on="click"
            :style="inputStyle"
            @update:value="handleConfirmPasswordChange"
            @keydown.enter="handlePasswordSubmit"
          />
        </n-form-item>

        <div class="mt-4 grid gap-3 rounded-[24px] border border-slate-200 bg-slate-50/80 p-5">
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-500">{{ $t('user.passwordConsistency') }}</span>
            <span :class="passwordMatch ? 'text-emerald-600' : 'text-amber-600'">
              {{ passwordMatch ? $t('user.passwordMatch') : $t('user.passwordNotMatch') }}
            </span>
          </div>
          <div class="flex items-center justify-between text-sm">
            <span class="text-slate-500">{{ $t('user.recommendStrength') }}</span>
            <span :class="passwordStrengthClass">{{ passwordStrengthText }}</span>
          </div>
        </div>

        <div class="mt-8 flex flex-col gap-3 sm:flex-row sm:justify-end">
          <n-button
            size="large"
            class="!h-12 !rounded-[18px] px-6"
            @click="resetForm"
          >{{ $t('commons.button.reset') }}</n-button>
          <n-button
            type="primary"
            size="large"
            class="!h-12 !rounded-[18px] px-8 shadow-[0_16px_30px_rgba(37,99,235,0.18)]"
            :disabled="!canSubmit"
            @click="handlePasswordSubmit"
          >
            {{ $t('commons.button.save') }}
          </n-button>
        </div>
      </n-form>
    </div>

    <!-- 修改个人资料弹窗 -->
    <n-modal
      v-model:show="showEditInfoModal"
      preset="card"
      title="修改个人资料"
      style="width: 480px;"
      class="w-full !rounded-[24px] shadow-[0_24px_48px_rgba(15,23,42,0.12)] sm:w-[90%]"
      @after-enter="initEditInfoForm"
    >
      <div class="mb-6 text-sm text-slate-500">更新您的控制台显示昵称和联系邮箱</div>
      <n-form
        ref="infoFormRef"
        :model="infoForm"
        :rules="infoRules"
        label-placement="left"
        label-width="80"
      >
        <n-form-item
          label="邮箱"
          path="email"
        >
          <n-input
            v-model:value="infoForm.email"
            placeholder="请输入您的邮箱地址"
            size="large"
            :style="inputStyle"
          />
        </n-form-item>
        <n-form-item
          label="昵称"
          path="nickName"
        >
          <n-input
            v-model:value="infoForm.nickName"
            placeholder="可选，用于显示在控制台"
            size="large"
            :style="inputStyle"
          />
        </n-form-item>
        <div class="mt-8 flex justify-end gap-3">
          <n-button
            size="large"
            class="!h-11 !rounded-[16px] px-6"
            @click="showEditInfoModal = false"
          >取消</n-button>
          <n-button
            type="primary"
            size="large"
            class="!h-11 !rounded-[16px] px-8 shadow-[0_12px_24px_rgba(37,99,235,0.18)]"
            :loading="savingInfo"
            @click="handleInfoSubmit"
          >保存修改</n-button>
        </div>
      </n-form>
    </n-modal>
  </div>
</template>

<script lang="ts" setup>
import { userEditPasswordAPI, userEditInfoAPI } from "@/api/modules/user"
import { useAuthStore } from "@/store/auth"
import { useMessage, NModal } from "naive-ui"
import { computed, ref, reactive } from "vue"
import { t } from "@/i18n"

const message = useMessage()
const authStore = useAuthStore()
const user = computed(() => authStore.user)

const password = ref("")
const newPassword = ref("")
const confirmPassword = ref("")

// 修改资料相关状态
const showEditInfoModal = ref(false)
const savingInfo = ref(false)
const infoFormRef = ref()
const infoForm = reactive({
	nickName: "",
	email: ""
})

const infoRules = {
	email: [
		{ required: true, message: "邮箱不能为空", trigger: "blur" } as any,
		{ type: "email", message: "请输入正确的邮箱格式，后续登录将使用该邮箱", trigger: ["blur", "input"] } as any
	]
}

const handlePasswordChange = (value: string) => {
	password.value = value
}

const handleNewPasswordChange = (value: string) => {
	newPassword.value = value
}

const handleConfirmPasswordChange = (value: string) => {
	confirmPassword.value = value
}

const inputStyle = {
	"--n-height": "56px",
	"--n-border-radius": "18px",
	"--n-padding-left": "18px",
	"--n-padding-right": "18px",
	"--n-font-size": "15px"
}

const passwordMatch = computed(
	() => Boolean(newPassword.value) && Boolean(confirmPassword.value) && newPassword.value === confirmPassword.value
)

const passwordStrengthText = computed(() => {
	const value = newPassword.value
	if (value.length >= 12 && /[A-Z]/.test(value) && /\d/.test(value) && /[^A-Za-z0-9]/.test(value)) {
		return t('user.strong')
	}
	if (value.length >= 8 && /[A-Za-z]/.test(value) && /\d/.test(value)) {
		return t('user.medium')
	}
	return t('user.weak')
})

const passwordStrengthClass = computed(() => {
	if (passwordStrengthText.value === t('user.strong')) {
		return "text-emerald-600"
	}
	if (passwordStrengthText.value === t('user.medium')) {
		return "text-blue-600"
	}
	return "text-amber-600"
})

const canSubmit = computed(
	() =>
		Boolean(password.value) &&
		Boolean(newPassword.value) &&
		Boolean(confirmPassword.value) &&
		passwordMatch.value &&
		newPassword.value.length >= 8
)

function resetForm() {
	password.value = ""
	newPassword.value = ""
	confirmPassword.value = ""
}

async function handlePasswordSubmit() {
	if (!password.value || !newPassword.value || !confirmPassword.value) {
		message.warning("请完整填写密码信息")
		return
	}
	if (newPassword.value !== confirmPassword.value) {
		message.error("两次输入的新密码不一致")
		return
	}
	if (newPassword.value.length < 8) {
		message.warning("新密码至少需要 8 位")
		return
	}
	try {
		await userEditPasswordAPI({
			password: password.value,
			newPassword: newPassword.value
		})
		message.success("密码修改成功")
		resetForm()
	} catch {
		message.error("密码修改失败")
	}
}

function initEditInfoForm() {
	infoForm.nickName = authStore.nickName || ""
	infoForm.email = user.value?.email || ""
}

async function handleInfoSubmit() {
	infoFormRef.value?.validate(async (errors: any) => {
		if (!errors) {
			savingInfo.value = true
			try {
				await userEditInfoAPI({
					nickName: infoForm.nickName,
					email: infoForm.email
				})
				message.success("资料修改成功")
				showEditInfoModal.value = false
				
				// 重新拉取最新的用户信息更新 Store
				await authStore.updateUser()
			} catch (error: any) {
			} finally {
				savingInfo.value = false
			}
		}
	})
}
</script>
