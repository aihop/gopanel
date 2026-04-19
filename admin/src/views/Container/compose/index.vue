<template>
  <div class="py-4">
    <!-- Header -->
    <div class="mb-4 flex items-center justify-between">
      <n-button
        type="primary"
        @click="openCreateModal"
      >创建编排</n-button>
      <n-space align="center">
        <n-popover
          trigger="click"
          placement="bottom-start"
          :width="300"
        >
          <template #trigger>
            <div class="rounded-full border border-gray-200 bg-base-100 p-2 px-5 text-sm">列表设置</div>
          </template>
          <div class="flex items-center gap-4 text-nowrap bg-base-100">
            刷新频率
            <n-select :options="[
								{ label: '不刷新', value: 0 },
								{ label: '5秒/次', value: 5 },
								{ label: '10秒/次', value: 10 },
								{ label: '30秒/次', value: 30 },
								{ label: '60秒/次', value: 60 },
								{ label: '120秒/次', value: 120 },
								{ label: '300秒/次', value: 300 }
							]" />
          </div>
        </n-popover>
        <n-input
          placeholder="搜索"
          clearable
          v-model:value="searchName"
          @keyup.enter="handleSearchEnter"
          style="width: 240px; border-radius: 30px; margin-right: 15px"
        >
          <template #suffix></template>
        </n-input>
      </n-space>
    </div>

    <!-- Card and Table -->
    <n-card title="编排">
      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="false"
        :bordered="false"
        class="mb-4"
      />
      <div class="flex items-center justify-end">
        <n-pagination
          v-model:page="paginationReactive.page"
          v-model:page-size="paginationReactive.pageSize"
          :item-count="paginationReactive.itemCount"
          :page-sizes="paginationReactive.pageSizes"
          show-size-picker
          show-quick-jumper
          @update:page="handlePageChange"
          @update:page-size="handlePageSizeChange"
        >
          <template #prefix="{ itemCount }">共 {{ itemCount }} 条</template>
        </n-pagination>
      </div>
    </n-card>
  </div>

  <n-drawer
    v-model:show="showCreateModal"
    :width="800"
    :mask-closable="false"
    placement="right"
  >
    <n-drawer-content
      title="创建编排"
      closable
    >
      <n-form
        ref="formRef"
        :model="composeForm"
      >
        <n-form-item
          :label="$t('database.source')"
          path="source"
        >
          <n-radio-group
            v-model:value="composeForm.source"
            name="composeSource"
          >
            <n-radio-button
              value="editor"
              label="编辑"
            />
            <n-radio-button
              value="path"
              label="路径选择"
            />
            <!-- <n-radio-button value="template" label="编排模版" /> -->
          </n-radio-group>
        </n-form-item>

        <div v-if="composeForm.source === 'editor'">
          <n-form-item
            label="文件夹"
            path="projectName"
          >
            <div style="display: flex; flex-direction: column; gap: 4px">
              <n-input
                v-model:value="composeForm.projectName"
                placeholder="文件夹"
              />
              <div style="font-size: 12px; margin-top: 4px; color: #adb0bc">
                配置文件保存路径: {{ baseDir
								}}{{ composeForm.projectName ? composeForm.projectName + "/" : "" }}
              </div>
            </div>
          </n-form-item>

          <n-tabs
            type="line"
            default-value="compose-definition"
            v-model:value="activeTab"
          >
            <n-tab-pane
              name="compose-definition"
              tab="编辑"
            >
              <FtEditor
                v-model="composeForm.composeContent"
                language="yaml"
                height="350px"
              />
            </n-tab-pane>
            <n-tab-pane
              name="compose-logs"
              tab="日志"
            >
              <div
                class="log-show"
                ref="logBoxRef"
              >
                {{ logContent }}
              </div>
            </n-tab-pane>
          </n-tabs>

          <div style="margin-top: 24px">
            <n-text strong>环境变量</n-text>
            <n-input
              style="margin-top: 8px"
              v-model:value="composeForm.envContent"
              type="textarea"
              :placeholder="envPlaceholder"
              :autosize="{ minRows: 4, maxRows: 8 }"
            />
            <div style="margin-top: 12px; font-size: 13px; line-height: 1.6">
              <n-text depth="3">
                注意: 设置的环境变量会写入
                <code>.env</code>
                文件 (位于项目目录下)。 默认会自动引用。
              </n-text>
            </div>
          </div>
        </div>

        <div v-if="composeForm.source === 'path'">
          <n-form-item
            label="编排文件路径"
            path="pathValue"
          >
            <n-input
              v-model:value="composeForm.pathValue"
              placeholder="例: /tmp/docker-compose.yml"
            />
          </n-form-item>
          <n-alert
            title="提示"
            type="info"
            :show-icon="false"
            style="margin-top: 10px; margin-bottom: 24px"
          >
            将从指定路径读取编排文件内容。环境变量文件 (如 .env ) 应位于同一目录下。
          </n-alert>

          <div style="margin-top: 24px">
            <n-text strong>环境变量</n-text>
            <n-input
              style="margin-top: 8px"
              v-model:value="composeForm.envContent"
              type="textarea"
              :placeholder="envPlaceholder"
              :autosize="{ minRows: 4, maxRows: 8 }"
            />
            <div style="margin-top: 12px; font-size: 13px; line-height: 1.6">
              <n-text depth="3">
                注意: 设置的环境变量会写入
                <code>.env</code>
                文件 (位于项目目录下)。 默认会自动引用。
              </n-text>
            </div>
          </div>
        </div>

        <!-- <div v-if="composeForm.source === 'template'">
					<n-form-item label="模版选择" path="selectedTemplateId">
						<n-select
							v-model:value="composeForm.selectedTemplateId"
							placeholder="请选择编排模版"
							:options="templateOptions"
						/>
					</n-form-item>
					<n-form-item label="Compose 名称" path="projectName">
						<div style="display: flex; flex-direction: column; gap: 4px">
							<n-input v-model:value="composeForm.projectName" placeholder="文件夹" />
							<div style="font-size: 12px; margin-top: 4px; color: #adb0bc">
								配置文件保存路径: {{ baseDir
								}}{{ composeForm.projectName ? composeForm.projectName + "/" : "" }}
							</div>
						</div>
					</n-form-item>

					<n-tabs type="line" default-value="compose-definition" v-model:value="activeTab">
						<n-tab-pane name="compose-definition" tab="编辑">
							<FtEditor v-model="composeForm.composeContent" language="yaml" />
						</n-tab-pane>
						<n-tab-pane name="compose-logs" tab="日志">
							<n-text depth="3">日志功能待后续添加。</n-text>
						</n-tab-pane>
					</n-tabs>

					<div style="margin-top: 24px">
						<n-text strong>环境变量</n-text>
						<n-input
							style="margin-top: 8px"
							v-model:value="composeForm.envContent"
							type="textarea"
							:placeholder="envPlaceholder"
							:autosize="{ minRows: 4, maxRows: 8 }"
						/>
						<div style="margin-top: 12px; font-size: 13px; line-height: 1.6">
							<n-text depth="3">
								注意: 设置的环境变量会写入
								<code>.env</code>
								文件 (位于项目目录下)。 默认会自动引用。
							</n-text>
						</div>
					</div>
				</div> -->
      </n-form>
      <template #footer>
        <n-button
          @click="showCreateModal = false"
          style="margin-right: 8px"
        >取消</n-button>
        <n-button
          type="primary"
          @click="handleConfirmCreate"
        >确认</n-button>
      </template>
    </n-drawer-content>
  </n-drawer>

  <n-modal
    v-model:show="showDeleteModal"
    preset="dialog"
    :title="'删除 - ' + deleteRow?.name"
  >
    <template #default>
      <n-checkbox v-model:checked="deleteWithFile">删除文件</n-checkbox>
      <div style="color: #888; margin: 8px 0 16px 0">
        删除容器编排的所有文件，包括配置文件和持久化文件，请谨慎操作！
      </div>
      <div style="color: #d03050; margin-bottom: 8px">
        删除操作无法回滚，请输入
        <b>"{{ deleteRow?.name }}"</b>
        删除此编排
      </div>
      <n-input
        v-model:value="deleteConfirmInput"
        placeholder="请输入名称"
      />
      <div
        v-if="deleteError"
        style="color: #d03050; margin-top: 8px"
      >{{ deleteError }}</div>
    </template>
    <template #action>
      <n-button @click="showDeleteModal = false">取消</n-button>
      <n-button
        type="error"
        @click="handleDeleteCompose"
        :disabled="deleteConfirmInput !== deleteRow?.name"
      >
        确认
      </n-button>
    </template>
  </n-modal>

  <EditDrawer
    ref="editDrawerRef"
    @search="search"
  />
</template>

<script setup lang="ts">
import { h, reactive, ref, onMounted, nextTick, watch } from "vue"
import {
	NButton,
	NDataTable,
	NPagination,
	NInput,
	NIcon,
	NSpace,
	NCard,
	type DataTableColumns,
	NDrawer,
	NDrawerContent,
	NForm,
	NFormItem,
	NRadioGroup,
	NRadioButton,
	NTabs,
	NTabPane,
	NText,
	NAlert,
	NPopover,
	NSelect,
	NModal,
	NCheckbox
} from "naive-ui"
import FtEditor from "@/components/FtEditor/index.vue"
import {
	searchCompose,
	composeOperator,
	testCompose,
	upCompose
} from "@/api/modules/container"
import { ReadByLine } from "@/api/modules/file"
import { appsGetBaseDir } from "@/api/modules/apps"
import EditDrawer from "./edit/index.vue"
import { t } from "@/i18n"
import { MsgError,MsgSuccess } from "@/utils/message"

type RowData = {
	key: number
	name: string
	source: string
	directory: string
	status: string
	createdTime: string
	containerNumber: number
	configFile: string
	workdir: string
	path: string
	containers: Array<{
		containerID: string
		name: string
		createTime: string
		state: string
	}>
	env: string | null
}

const data = ref<RowData[]>([])

const loading = ref(false)
const paginationReactive = reactive({
	page: 1,
	pageSize: 10,
	itemCount: 0,
	showSizePicker: true,
	pageSizes: [10, 20, 50, 100],
	onChange: (page: number) => {
		paginationReactive.page = page
		search()
	},
	onUpdatePageSize: (pageSize: number) => {
		paginationReactive.pageSize = pageSize
		paginationReactive.page = 1
		search()
	}
})
const searchName = ref()

const search = async () => {
	const params: any = {
		page: paginationReactive.page,
		pageSize: paginationReactive.pageSize
	}
	if (searchName.value && String(searchName.value).trim() !== "") {
		params.info = String(searchName.value).trim()
	} else {
		params.info = ""
	}
	loading.value = true
	await searchCompose(params)
		.then(res => {
			loading.value = false
			if (res.data && res.data.items) {
				tableData.value = res.data.items.map((item: any, index: number) => ({
					key: index,
					name: item.name,
					source: item.createdBy || "自定义", // 使用createdBy字段作为来源
					directory: item.workdir,
					status:
						item.containers && Array.isArray(item.containers)
							? `${item.containers.filter((c: any) => c.state === "running").length}/${item.containerNumber}`
							: "0/0",
					createdTime: item.createdAt,
					containerNumber: item.containerNumber,
					configFile: item.configFile,
					workdir: item.workdir,
					path: item.path,
					containers: item.containers || [],
					env: item.env
				}))
				paginationReactive.itemCount = res.data.total
			}
		})
		.finally(() => {
			loading.value = false
		})
}

const baseDir = ref("")

onMounted(async () => {
	try {
		const res = await appsGetBaseDir()
		if (res.code == 0) {
			baseDir.value = res.data || "/opt/gopanel/data/docker/compose/"
		}
	} catch {
		baseDir.value = "/opt/gopanel/data/docker/compose/"
	}
	search()
})

const createColumns = ({
	edit,
	remove
}: {
	edit: (row: RowData) => void
	remove: (row: RowData) => void
}): DataTableColumns<RowData> => {
	return [
		{
			title: t("commons.table.name"),
			key: "name",
			render(row) {
				return h("a", { onClick: () => console.log("Clicked name: " + row.name) }, row.name)
			}
		},
		{
			title: t("database.source"),
			key: "source"
		},
		{
			title: t("container.composeDirectory"),
			key: "directory",
			align: "center"
		},
		{
			title: t("container.containerStatus"),
			key: "status"
		},
		{
			title: t("commons.table.createdAt"),
			key: "createdTime"
		},
		{
			title: t("database.actions"),
			key: "actions",
			render(row) {
				return h(
					NSpace,
					{},
					{
						default: () => [
							h(
								NButton,
								{
									text: true,
									tertiary: true,
									type: "primary",
									onClick: () => edit(row)
								},
								{ default: () => "编辑" }
							),
							h(
								NButton,
								{
									text: true,
									tertiary: true,
									type: "error",
									onClick: () => remove(row)
								},
								{ default: () => "删除" }
							)
						]
					}
				)
			}
		}
	]
}

const tableData = ref<RowData[]>([])

const columns = createColumns({
	edit: handleEdit,
	remove: (row: RowData) => {
		openDeleteModal(row)
	}
})
 
const showCreateModal = ref(false)

const initialComposeFormState = {
	source: "editor", // 'editor', 'path', 'template'
	projectName: "",
	composeContent: "",
	envContent: "",
	pathValue: "", // for source 'path'
	selectedTemplateId: null // Added for template selection
}

const composeForm = reactive({ ...initialComposeFormState })

const envPlaceholder = "一行一个, 例:\nkey1=value1\nkey2=value2"

const templateOptions = ref([
	{ label: "基础 Web 应用 (Nginx + PHP)", value: "web-php" },
	{ label: "数据库服务 (MySQL)", value: "mysql-db" },
	{ label: "缓存服务 (Redis)", value: "redis-cache" }
])

const activeTab = ref("compose-definition")
const logContent = ref("")
const logLoading = ref(false)
let logTimer: any = null
const logBoxRef = ref<HTMLElement | null>(null)

const openCreateModal = () => {
	Object.assign(composeForm, initialComposeFormState)
	composeForm.composeContent = "" // Ensure placeholder is set
	composeForm.selectedTemplateId = null // Reset template selection
	showCreateModal.value = true
}

const handleConfirmCreate = async () => {
	if (!composeForm.projectName || composeForm.projectName.trim() === "") {
		MsgError("请填写文件夹名称")
		return
	}
	if (!/^[a-zA-Z0-9_-]+$/.test(composeForm.projectName)) {
		MsgError("文件夹名称错误")
		return
	}
	// 组装参数
	const envArr = (composeForm.envContent || "")
		.split("\n")
		.map(line => line.trim())
		.filter(Boolean)
	const envStr = envArr.join("\n")
	const params: any = {
		name: composeForm.projectName,
		from: composeForm.source === "editor" ? "edit" : composeForm.source,
		path: composeForm.source === "path" ? composeForm.pathValue : "",
		file: composeForm.composeContent,
		env: envArr,
		envStr: envStr,
		envFileContent: composeForm.envContent
	}
	if (composeForm.selectedTemplateId) {
		params.template = composeForm.selectedTemplateId
	}
	try {
		// 1. 先测试
		const testRes = await testCompose(params)
		if (testRes.code !== 200 && testRes.code !== 0) {
			MsgError(testRes.message || "配置测试失败")
			return
		}
		// 2. 创建
		const createRes = await upCompose(params)
		if (createRes.code !== 200 && createRes.code !== 0) {
			MsgError(createRes.message || "创建失败")
			return
		}
		// 3. 获取日志文件名，切换tab并轮询日志
		nextTick(() => {
			activeTab.value = "compose-logs"
		})
		const logFile = createRes.data
		// activeTab.value = "compose-logs"
		logContent.value = ""
		logLoading.value = true
		if (logTimer) clearInterval(logTimer)
		const fetchLog = async () => {
			try {
				const res = await ReadByLine({ type: "compose-create", name: logFile, page: 1, pageSize: 1000 })
				// 根据接口返回内容判断是否停止轮询
				if (res.data && res.data.lines) {
					logContent.value = res.data.lines.join("\n")
					const lastLine = res.data.lines[res.data.lines.length - 1] || ""
					if (lastLine.includes("successful")) {
						clearInterval(logTimer)
					}
				}
			} catch {}
		}
		await fetchLog()
		logTimer = setInterval(fetchLog, 2000)
		MsgSuccess("创建任务已提交，正在获取日志...")
	} catch (e: any) {
		MsgError(e?.message)
	}
}

const showDeleteModal = ref(false)
const deleteRow = ref<RowData | null>(null)
const deleteWithFile = ref(false)
const deleteConfirmInput = ref("")
const deleteError = ref("")

function openDeleteModal(row: RowData) {
	deleteRow.value = row
	deleteWithFile.value = false
	deleteConfirmInput.value = ""
	deleteError.value = ""
	showDeleteModal.value = true
}

async function handleDeleteCompose() {
	if (!deleteRow.value) return
	if (deleteConfirmInput.value !== deleteRow.value.name) {
		deleteError.value = "请输入正确的名称以确认删除"
		return
	}
	deleteError.value = ""
	try {
		await composeOperator({
			name: deleteRow.value.name,
			path: deleteRow.value.path,
			operation: "delete",
			withFile: deleteWithFile.value
		})
		showDeleteModal.value = false
		// 删除后刷新列表
		search()
	} catch (e) {
		deleteError.value = "删除失败，请重试"
	}
}

function handlePageChange(page: number) {
	paginationReactive.page = page
	search()
}
function handlePageSizeChange(pageSize: number) {
	paginationReactive.pageSize = pageSize
	paginationReactive.page = 1
	search()
}

// 搜索输入框回车事件
function handleSearchEnter() {
	paginationReactive.page = 1
	search()
}

const scrollLogToBottom = () => {
	nextTick(() => {
		if (logBoxRef.value) {
			logBoxRef.value.scrollTop = logBoxRef.value.scrollHeight
		}
	})
}

watch(logContent, () => {
	scrollLogToBottom()
})

const editDrawerRef = ref()

function handleEdit(row: RowData) {
	if (editDrawerRef.value && editDrawerRef.value.acceptParams) {
		editDrawerRef.value.acceptParams({
			name: row.name,
			path: row.path,
			content: row.configFile,
			env: row.env ? row.env.split("\n") : [],
			envStr: row.env || "",
			createdBy: row.source
		})
	}
}
</script>

<style scoped>
.n-form-item-feedback-wrapper {
	min-height: auto !important; /* Override to prevent extra space if feedback is short */
}
.log-show {
	background: #181818;
	color: #e0e0e0;
	padding: 12px;
	border-radius: 4px;
	min-height: 350px;
	max-height: 400px;
	overflow: auto;
	font-family: monospace;
	font-size: 13px;
	white-space: pre-line;
}
</style>
