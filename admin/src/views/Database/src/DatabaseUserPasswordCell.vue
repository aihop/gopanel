<script setup lang="ts">
import { ref } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { databaseUserPasswordAPI } from "@/api/modules/database"

const props = defineProps<{
	id: number
	serverId: number
	username: string
	host: string
	managed: boolean
}>()

const { t } = useI18n()
const message = useMessage()
const password = ref<string | null>(null)
const loading = ref(false)

const revealPassword = async () => {
	if (password.value !== null || loading.value) return
	loading.value = true
	try {
		const request = props.id
			? { id: props.id }
			: { serverId: props.serverId, username: props.username, host: props.host }
		const { data } = await databaseUserPasswordAPI(request)
		if (!data?.managed) {
			message.warning(t("database.passwordNotManaged"))
			return
		}
		password.value = data.password
	} catch (_error) {
		message.error(t("database.passwordLoadFailed"))
	} finally {
		loading.value = false
	}
}

const copyPassword = async () => {
	if (password.value === null) await revealPassword()
	if (password.value === null) return
	try {
		await navigator.clipboard.writeText(password.value)
		message.success(t("database.passwordCopied"))
	} catch (_error) {
		message.error(t("database.copyFailed"))
	}
}
</script>

<template>
  <n-tag
    v-if="!managed"
    type="warning"
  >
    {{ $t("database.passwordNotManaged") }}
  </n-tag>
  <n-flex
    v-else
    align="center"
    :size="6"
  >
    <n-input
      size="small"
      :value="password === null ? '••••••••' : password"
      readonly
      style="width: 135px"
    />
    <n-button
      size="tiny"
      :loading="loading"
      @click="password === null ? revealPassword() : (password = null)"
    >
      {{ password === null ? $t("database.showPassword") : $t("database.hidePassword") }}
    </n-button>
    <n-button
      v-if="password !== null"
      size="tiny"
      @click="copyPassword"
    >
      {{ $t("database.copy") }}
    </n-button>
  </n-flex>
</template>
