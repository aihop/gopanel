<template>
	<div class="mt-4">
		<n-card :bordered="false" class="rounded-xl shadow-sm">
			<div class="mb-4 flex items-center justify-between">
				<div>
					<h3 class="text-lg font-medium text-slate-800">{{ t("setting.adminAndSubAccounts") }}</h3>
					<p class="mt-1 text-sm text-slate-500">{{ t("setting.adminHelper") }}</p>
				</div>
				<n-button type="primary" @click="openModal()">
					{{ t("setting.addAdmin") }}
				</n-button>
			</div>

			<n-data-table
				:columns="columns"
				:data="dataList"
				:loading="loading"
				:pagination="pagination"
				:bordered="false"
				size="small"
			/>
		</n-card>

		<!-- 新增/编辑弹窗 -->
		<n-modal v-model:show="modalVisible" preset="card" :title="modalTitle" style="width: 700px">
			<n-form
				ref="formRef"
				:model="form"
				:rules="rules"
				label-placement="top"
				require-mark-placement="right-hanging"
			>
				<n-form-item :label="t('setting.loginEmail')" path="email">
					<n-input
						v-model:value="form.email"
						:placeholder="t('setting.loginEmailPlaceholder')"
						:disabled="!!form.id"
					/>
				</n-form-item>

				<n-form-item :label="t('setting.nickname')" path="nickName">
					<n-input v-model:value="form.nickName" :placeholder="t('setting.nicknamePlaceholder')" />
				</n-form-item>

				<n-form-item :label="t('setting.loginPassword')" path="password">
					<n-input
						v-model:value="form.password"
						type="password"
						show-password-on="click"
						:placeholder="t('setting.passwordPlaceholder')"
					/>
				</n-form-item>

				<n-form-item :label="t('setting.rolePermission')" path="role">
					<n-select v-model:value="form.role" :options="roleOptions" />
				</n-form-item>

				<template v-if="form.role === 'SUB_ADMIN' || form.role === 'DEMO'">
					<n-form-item :label="t('setting.menuPermission')" path="menus">
						<n-tree-select
							v-model:value="form.menus"
							multiple
							cascade
							checkable
							:options="menuOptions"
							:placeholder="t('setting.menuPermissionPlaceholder')"
						/>
						<template #feedback>
							{{ t("setting.menuPermissionFeedback") }}
						</template>
					</n-form-item>

					<n-form-item :label="t('setting.fileBaseDirLabel')" path="fileBaseDir">
						<n-input v-model:value="form.fileBaseDir" :placeholder="t('setting.fileBaseDirPlaceholder')" />
						<template #feedback>
							{{ t("setting.fileBaseDirFeedback") }}
						</template>
					</n-form-item>
				</template>
			</n-form>

			<template #footer>
				<div class="flex justify-end gap-2">
					<n-button @click="modalVisible = false">{{ t("commons.button.cancel") }}</n-button>
					<n-button type="primary" :loading="submitting" @click="handleSubmit">
						{{ t("commons.button.save") }}
					</n-button>
				</div>
			</template>
		</n-modal>
	</div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from "vue"
import {
	NCard,
	NButton,
	NDataTable,
	NModal,
	NForm,
	NFormItem,
	NInput,
	NSelect,
	NTreeSelect,
	NTag,
	NPopconfirm,
	useMessage
} from "naive-ui"
import { createUserAPI, updateUserAPI, deleteUserAPI, pageUserAPI } from "@/api/modules/user"
import { t } from "@/i18n"

const message = useMessage()

const loading = ref(false)
const submitting = ref(false)
const dataList = ref([])
const pagination = reactive({
	page: 1,
	limit: 10,
	itemCount: 0,
	onChange: (page: number) => {
		pagination.page = page
		fetchData()
	}
})

const modalVisible = ref(false)
const modalTitle = ref(t("setting.addAdmin"))
const formRef = ref(null)

const form = reactive({
	id: 0,
	email: "",
	nickName: "",
	password: "",
	role: "SUB_ADMIN",
	fileBaseDir: "",
	menus: [] as string[]
})

const rules = {
	email: [{ required: true, message: t("setting.loginEmailRequired"), trigger: "blur" }],
	nickName: [{ required: true, message: t("setting.nicknameRequired"), trigger: "blur" }],
	role: [{ required: true, message: t("setting.roleRequired"), trigger: "blur" }]
}

const roleOptions = [
	{ label: t("setting.roleSuperAdmin"), value: "ADMIN" },
	{ label: t("setting.roleSubAdmin"), value: "SUB_ADMIN" },
	{ label: t("setting.roleDemo"), value: "DEMO" }
]

const menuOptions = [
	{ label: t("menu.code"), key: "code" },
	{ label: t("menu.flow"), key: "flow" },
	{ label: t("menu.dashboard"), key: "dashboard" },
	{ label: t("menu.website"), key: "website" },
	{ label: t("menu.ssl"), key: "ssl" },
	{ label: t("menu.database"), key: "database" },
	{ label: t("menu.container"), key: "container" },
	{ label: t("menu.pipeline"), key: "pipeline" },
	{ label: t("menu.apps"), key: "apps" },
	{
		label: t("menu.host"),
		key: "host",
		children: [
			{ label: t("menu.files"), key: "Host-Files" },
			{ label: t("menu.firewall"), key: "Host-Firewall" },
			{ label: t("menu.processManage"), key: "Host-Process" },
			{ label: t("menu.daemon"), key: "Toolbox-Daemon" },
			{ label: t("menu.monitor"), key: "Host-Monitor" }
		]
	},
	{ label: t("menu.setting"), key: "setting" }
]

const columns = [
	{ title: "ID", key: "id", width: 60 },
	{ title: t("setting.nickname"), key: "nickName" },
	{ title: t("setting.email"), key: "email" },
	{
		title: t("setting.role"),
		key: "role",
		render(row: any) {
			if (row.role === "ADMIN" || row.role === "SUPER") {
				return h(
					NTag,
					{ type: "success", size: "small", bordered: false },
					{ default: () => t("setting.roleSuperAdminShort") }
				)
			} else if (row.role === "DEMO") {
				return h(
					NTag,
					{ type: "info", size: "small", bordered: false },
					{ default: () => t("setting.roleDemo") }
				)
			} else {
				return h(
					NTag,
					{ type: "warning", size: "small", bordered: false },
					{ default: () => t("setting.roleSubAdminShort") }
				)
			}
		}
	},
	{
		title: t("setting.fileBaseDirTitle"),
		key: "fileBaseDir",
		render(row: any) {
			if (row.role === "ADMIN" || row.role === "SUPER") {
				return h("span", { class: "text-slate-400" }, t("setting.noRestriction"))
			}
			return row.fileBaseDir
				? h("code", { class: "bg-slate-100 px-1 py-0.5 rounded text-xs text-slate-600" }, row.fileBaseDir)
				: h("span", { class: "text-slate-400" }, t("setting.notSetDefault"))
		}
	},
	{
		title: t("commons.table.operate"),
		key: "actions",
		width: 150,
		render(row: any) {
			if (row.role === "SUPER") {
				return h("span", { class: "text-slate-400 text-xs" }, t("setting.cannotModifyCreator"))
			}
			return h("div", { class: "flex gap-2" }, [
				h(
					NButton,
					{ size: "small", type: "primary", ghost: true, onClick: () => openModal(row) },
					{ default: () => t("commons.button.edit") }
				),
				h(
					NPopconfirm,
					{ onPositiveClick: () => handleDelete(row.id) },
					{
						trigger: () =>
							h(
								NButton,
								{ size: "small", type: "error", ghost: true },
								{ default: () => t("commons.button.delete") }
							),
						default: () => t("setting.confirmDeleteAdmin")
					}
				)
			])
		}
	}
]

async function fetchData() {
	loading.value = true
	try {
		const res = await pageUserAPI({
			page: pagination.page,
			limit: pagination.limit
		})
		const responseData = res.data as any
		dataList.value = responseData.items || []
		pagination.itemCount = responseData.total || 0
	} catch (error: any) {
		void 0
	} finally {
		loading.value = false
	}
}

function openModal(row?: any) {
	if (row) {
		modalTitle.value = t("setting.editAdmin")
		form.id = row.id
		form.email = row.email
		form.nickName = row.nickName
		form.password = ""
		form.role = row.role
		form.fileBaseDir = row.fileBaseDir || ""
		form.menus = row.menus ? row.menus.split(",").filter((m: string) => m.trim() !== "") : []
	} else {
		modalTitle.value = t("setting.addAdmin")
		form.id = 0
		form.email = ""
		form.nickName = ""
		form.password = ""
		form.role = "SUB_ADMIN"
		form.fileBaseDir = ""
		// 默认全选
		const allKeys: string[] = []
		menuOptions.forEach(m => {
			allKeys.push(m.key)
			if (m.children) {
				m.children.forEach(c => allKeys.push(c.key))
			}
		})
		form.menus = allKeys
	}
	modalVisible.value = true
}

async function handleSubmit() {
	;(formRef.value as any)?.validate(async (errors: any) => {
		if (!errors) {
			if (!form.id && !form.password) {
				message.error(t("setting.initialPasswordRequired"))
				return
			}

			submitting.value = true
			const submitData = {
				...form,
				menus: form.menus.join(",")
			}

			try {
				if (form.id) {
					await updateUserAPI(submitData)
					message.success(t("setting.editSuccess"))
				} else {
					await createUserAPI(submitData)
					message.success(t("setting.createSuccess"))
				}
				modalVisible.value = false
				fetchData()
			} catch (error: any) {
				void 0
			} finally {
				submitting.value = false
			}
		}
	})
}

async function handleDelete(id: number) {
	try {
		await deleteUserAPI({ id })
		message.success(t("setting.deleteSuccess"))
		fetchData()
	} catch (error: any) {
		void 0
	}
}

onMounted(() => {
	fetchData()
})
</script>
