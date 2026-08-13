<template>
  <div class="mt-4">
    <n-card
      :bordered="false"
      class="rounded-xl shadow-sm"
    >
      <div class="mb-4 flex items-center justify-between">
        <div>
          <h3 class="text-lg font-medium text-slate-800">管理员与子账户</h3>
          <p class="text-sm text-slate-500 mt-1">设置多账户登录，配置普通管理员的文件访问目录以隔离权限。</p>
        </div>
        <n-button
          type="primary"
          @click="openModal()"
        >
          添加管理员
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
    <n-modal
      v-model:show="modalVisible"
      preset="card"
      :title="modalTitle"
      style="width: 700px;"
    >
      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-placement="top"
        require-mark-placement="right-hanging"
      >
        <n-form-item
          label="登录邮箱"
          path="email"
        >
          <n-input
            v-model:value="form.email"
            placeholder="请输入用于登录的邮箱"
            :disabled="!!form.id"
          />
        </n-form-item>

        <n-form-item
          label="昵称"
          path="nickName"
        >
          <n-input
            v-model:value="form.nickName"
            placeholder="请输入管理员昵称"
          />
        </n-form-item>

        <n-form-item
          label="登录密码"
          path="password"
        >
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="请输入密码 (不少于6位，留空表示不修改)"
          />
        </n-form-item>

        <n-form-item
          label="角色权限"
          path="role"
        >
          <n-select
            v-model:value="form.role"
            :options="roleOptions"
          />
        </n-form-item>

        <template v-if="form.role === 'SUB_ADMIN' || form.role === 'DEMO'">
          <n-form-item
            label="面板菜单权限"
            path="menus"
          >
            <n-tree-select
              v-model:value="form.menus"
              multiple
              cascade
              checkable
              :options="menuOptions"
              placeholder="请选择允许访问的菜单模块"
            />
            <template #feedback>
              未勾选的菜单项将不会在左侧导航栏中显示，并且禁止访问该路由。
            </template>
          </n-form-item>

          <n-form-item
            label="文件管理限制目录"
            path="fileBaseDir"
          >
            <n-input
              v-model:value="form.fileBaseDir"
              placeholder="例如：/var/www/my_project"
            />
            <template #feedback>
              该管理员在文件管理中只能访问此目录下的文件，且无法使用终端(Terminal)。
            </template>
          </n-form-item>
        </template>
      </n-form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <n-button @click="modalVisible = false">取消</n-button>
          <n-button
            type="primary"
            :loading="submitting"
            @click="handleSubmit"
          >保存</n-button>
        </div>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from "vue"
import { NCard, NButton, NDataTable, NModal, NForm, NFormItem, NInput, NSelect, NTreeSelect, NTag, NPopconfirm, useMessage } from "naive-ui"
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
const modalTitle = ref("添加管理员")
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
	email: [{ required: true, message: "请输入登录邮箱", trigger: "blur" }],
	nickName: [{ required: true, message: "请输入昵称", trigger: "blur" }],
	role: [{ required: true, message: "请选择角色", trigger: "blur" }]
}

const roleOptions = [
	{ label: "超级管理员 (拥有所有权限)", value: "ADMIN" },
	{ label: "普通管理员 (可限制文件目录)", value: "SUB_ADMIN" }, 
	{ label: "演示权限", value: "DEMO" }
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
		label: "主机", 
		key: "host",
		children: [
			{ label: "文件管理", key: "Host-Files" },
			{ label: "防火墙", key: "Host-Firewall" },
			{ label: "进程管理", key: "Host-Process" },
			{ label: "守护进程", key: "Toolbox-Daemon" },
			{ label: "资源监控", key: "Host-Monitor" }
		]
	},
	{ label: "系统设置", key: "setting" }
]

const columns = [
	{ title: "ID", key: "id", width: 60 },
	{ title: "昵称", key: "nickName" },
	{ title: "邮箱", key: "email" },
	{ 
		title: "角色", 
		key: "role",
		render(row: any) {
			if (row.role === 'ADMIN' || row.role === 'SUPER') {
				return h(NTag, { type: 'success', size: 'small', bordered: false }, { default: () => '超级管理员' })
			} else if (row.role === 'DEMO') {
				return h(NTag, { type: 'info', size: 'small', bordered: false }, { default: () => '演示权限' })
			} else {
				return h(NTag, { type: 'warning', size: 'small', bordered: false }, { default: () => '普通管理员' })
			}
		}
	},
	{ 
		title: "文件目录限制", 
		key: "fileBaseDir",
		render(row: any) {
			if (row.role === 'ADMIN' || row.role === 'SUPER') {
				return h('span', { class: 'text-slate-400' }, '无限制 ( / )')
			}
			return row.fileBaseDir ? h('code', { class: 'bg-slate-100 px-1 py-0.5 rounded text-xs text-slate-600' }, row.fileBaseDir) : h('span', { class: 'text-slate-400' }, '未设置 (默认 /)')
		}
	},
	{
		title: "操作",
		key: "actions",
		width: 150,
		render(row: any) {
			if (row.role === 'SUPER') {
				return h('span', { class: 'text-slate-400 text-xs' }, '不可修改系统创建者')
			}
			return h("div", { class: "flex gap-2" }, [
				h(NButton, { size: "small", type: "primary", ghost: true, onClick: () => openModal(row) }, { default: () => "编辑" }),
				h(
					NPopconfirm,
					{ onPositiveClick: () => handleDelete(row.id) },
					{
						trigger: () => h(NButton, { size: "small", type: "error", ghost: true }, { default: () => "删除" }),
						default: () => "确认删除该账号？删除后无法恢复。"
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
		modalTitle.value = "编辑管理员"
		form.id = row.id
		form.email = row.email
		form.nickName = row.nickName
		form.password = ""
		form.role = row.role
		form.fileBaseDir = row.fileBaseDir || ""
		form.menus = row.menus ? row.menus.split(",").filter((m: string) => m.trim() !== "") : []
	} else {
		modalTitle.value = "添加管理员"
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
	(formRef.value as any)?.validate(async (errors: any) => {
		if (!errors) {
			if (!form.id && !form.password) {
				message.error("新建账号必须设置初始密码")
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
					message.success("修改成功")
				} else {
					await createUserAPI(submitData)
					message.success("创建成功")
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
		message.success("删除成功")
		fetchData()
	} catch (error: any) {
		void 0
	}
}

onMounted(() => {
	fetchData()
})
</script>
