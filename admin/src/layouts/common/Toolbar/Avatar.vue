<template>
	<n-dropdown :options="options" placement="bottom-end" @select="handleSelect">
		<n-avatar round :size="32" :src="authStore.userAvatar" :img-props="{ alt: 'avatar' }" class="p-2" />
	</n-dropdown>
</template>

<script lang="ts" setup>
import { renderIcon } from "@/utils"
import { NAvatar, NDropdown, useDialog } from "naive-ui"
import { useAuthStore } from "@/store/auth"
import { useRouter } from "vue-router"
import { useMessage } from "naive-ui"

const authStore = useAuthStore()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const { inner } = defineProps({
	inner: {
		type: Boolean,
		default: true
	}
})

const { t } = useI18n()

const UserIcon = "ion:person-outline"
const LogoutIcon = "ion:log-out-outline"

const options = computed(() => {
	const menuItems = [
		inner && {
			label: t("user.profile"),
			key: "route-Profile",
			icon: renderIcon(UserIcon)
		},
		{
			label: t("auth.logout"),
			key: "logout",
			icon: renderIcon(LogoutIcon)
		}
	]

	return menuItems.filter(item => !!item)
})

async function handleSelect(key: string) {
	if (key === "logout") {
		// 显示确认对话框
		dialog.info({
			title: t("commons.msg.sureLogOut"),
			positiveText: t("commons.button.confirm"),
			negativeText: t("commons.button.cancel"),
			onPositiveClick: async () => {
				authStore.setLogout()
				router.push("/login")
				message.success(t("auth.logout"))
			}
		})
	} else if (key.indexOf("route-") === 0) {
		const path = key.split("route-")[1]
		router.push({ name: path })
	}
}
</script>
