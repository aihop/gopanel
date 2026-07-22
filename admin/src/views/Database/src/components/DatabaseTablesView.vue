<script setup lang="ts">
import { ref, computed, h, onMounted, watch } from 'vue'
import { useMessage, useDialog, NEmpty, NIcon, NButton, NDataTable, NPagination, NPopconfirm, NCheckbox, NTag, NInput, NModal, NPopselect } from 'naive-ui'
import { renderIcon } from '@/utils'
import { getDBManagerTableListAPI, execDBManagerSqlAPI, chunkImportDBAPI, databaseUserListAPI, changeDBManagerTableOwnerAPI } from '@/api/modules/database'

const props = defineProps<{
  selectedServerId: number | null
  selectedDatabase: string | null
  serverOptions: any[]
}>()

const emit = defineEmits<{
  (e: 'selectTable', tableName: string): void
}>()

const message = useMessage()
const dialog = useDialog()
const loading = ref(false)
const tables = ref<any[]>([])
const total = ref(0)
const page = ref(1)
const limit = ref(20)
const checkedKeys = ref<string[]>([])
const importing = ref(false)
const importProgress = ref(0)
const fileInputRef = ref<HTMLInputElement | null>(null)
const batchLoading = ref(false)

// 视图管理
const viewMode = ref<'tables' | 'views' | 'routines'>('tables')
const views = ref<string[]>([])
const loadingViews = ref(false)
const viewDefinition = ref('')
const viewDefinitionTitle = ref('')
const showViewDefModal = ref(false)

// 函数/存储过程管理
const routines = ref<{ name: string; type: string }[]>([])
const loadingRoutines = ref(false)

// 所属用户（MySQL/MariaDB 按库授权，Postgres 有真正的表级 owner）
const dbUsers = ref<{ username: string; privileges: string[] }[]>([])
const changingOwnerTable = ref<string | null>(null)

const authorizedUsersForDb = computed(() => {
  if (!props.selectedDatabase) return []
  return dbUsers.value.filter(u => u.privileges?.includes(props.selectedDatabase!)).map(u => u.username)
})

const ownerOptions = computed(() => dbUsers.value.map(u => ({ label: u.username, value: u.username })))

const fetchDbUsers = async () => {
  if (!props.selectedServerId) { dbUsers.value = []; return }
  try {
    const res: any = await databaseUserListAPI({
      wheres: [{ field: 'server_id', rule: 'eq', val: String(props.selectedServerId) }]
    })
    if (res.code === 0 && res.data) {
      dbUsers.value = (res.data.items || []).map((u: any) => ({ username: u.username, privileges: u.privileges || [] }))
    } else {
      dbUsers.value = []
    }
  } catch {
    dbUsers.value = []
  }
}

const handleChangeOwner = async (tableName: string, owner: string) => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  changingOwnerTable.value = tableName
  try {
    const res: any = await changeDBManagerTableOwnerAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      tableName,
      owner
    })
    if (res.code === 0) {
      message.success(`表 ${tableName} 的所有者已改为 ${owner}`)
      fetchData()
    } else {
      message.error(res.message || '修改失败')
    }
  } catch {
    message.error('修改请求失败')
  } finally {
    changingOwnerTable.value = null
  }
}

const serverType = computed(() => {
  const s = props.serverOptions.find(s => s.value === props.selectedServerId)
  return s?.type || ''
})

const formatSize = (bytes: number) => {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) { size /= 1024; i++ }
  return `${size.toFixed(1)} ${units[i]}`
}

const formatCount = (n: number) => {
  if (n === null || n === undefined) return '-'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return String(n)
}

const columns = [
  {
    type: 'selection' as const,
    width: 40,
    disabled: () => false
  },
  {
    title: '表名',
    key: 'name',
    width: 200,
    render: (row: any) => {
      return h('span', {
        class: 'cursor-pointer text-blue-600 hover:text-blue-800 font-medium',
        onClick: () => emit('selectTable', row.name)
      }, row.name)
    }
  },
  {
    title: '引擎',
    key: 'engine',
    width: 100,
    render: (row: any) => row.engine || '-'
  },
  {
    title: '行数',
    key: 'rowCount',
    width: 100,
    sorter: true,
    render: (row: any) => formatCount(row.rowCount)
  },
  {
    title: '大小',
    key: 'sizeBytes',
    width: 110,
    sorter: true,
    render: (row: any) => formatSize(row.sizeBytes)
  },
  {
    title: '字符集',
    key: 'collation',
    width: 130,
    render: (row: any) => row.collation || '-'
  },
  {
    title: '创建时间',
    key: 'createTime',
    width: 160,
    render: (row: any) => row.createTime || '-'
  },
  {
    title: '更新时间',
    key: 'updateTime',
    width: 160,
    render: (row: any) => row.updateTime || '-'
  },
  {
    title: '所属用户',
    key: 'owner',
    width: 180,
    render: (row: any) => {
      // Postgres 有真正的表级 owner，可以直接改
      if (serverType.value === 'postgresql') {
        return h(
          NPopselect,
          {
            options: ownerOptions.value,
            value: row.owner,
            trigger: 'click',
            'onUpdate:value': (val: string) => handleChangeOwner(row.name, val)
          },
          {
            default: () =>
              h(
                NTag,
                { size: 'small', type: 'info', bordered: false, class: 'cursor-pointer' },
                { default: () => (changingOwnerTable.value === row.name ? '修改中...' : row.owner || '-') }
              )
          }
        )
      }
      // MySQL/MariaDB 是按库整体授权的，没有表级归属，只能展示当前库被授权给了谁
      if (serverType.value === 'mysql' || serverType.value === 'mariadb') {
        const users = authorizedUsersForDb.value
        if (users.length === 0) return h('span', { class: 'text-gray-400' }, '-')
        return h(
          'span',
          { title: '该数据库类型按库授权，非按表，此处显示的是当前库的授权用户' },
          users.join(', ')
        )
      }
      return h('span', { class: 'text-gray-400' }, '-')
    }
  }
]

const handleImportSQL = () => {
  fileInputRef.value?.click()
}

const CHUNK_IMPORT_SIZE = 1024 * 1024 // 1MB 每分片

const onFileSelected = async (e: Event) => {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  if (!props.selectedServerId || !props.selectedDatabase) {
    message.warning('请先选择数据库服务器和数据库')
    target.value = ''
    return
  }
  if (!file.name.toLowerCase().endsWith('.sql')) {
    message.error('请选择 .sql 文件')
    target.value = ''
    return
  }

  importing.value = true
  importProgress.value = 0
  const fileSize = file.size
  const chunkCount = Math.ceil(fileSize / CHUNK_IMPORT_SIZE)
  let uploadedChunks = 0

  try {
    for (let i = 0; i < chunkCount; i++) {
      const start = i * CHUNK_IMPORT_SIZE
      const end = Math.min(start + CHUNK_IMPORT_SIZE, fileSize)
      const chunk = file.slice(start, end)

      const fd = new FormData()
      fd.append('serverId', String(props.selectedServerId))
      fd.append('databaseName', props.selectedDatabase!)
      fd.append('filename', file.name)
      fd.append('chunk', chunk)
      fd.append('chunkIndex', i.toString())
      fd.append('chunkCount', chunkCount.toString())

      const res = await chunkImportDBAPI(fd)
      uploadedChunks++
      importProgress.value = Math.round((uploadedChunks / chunkCount) * 100)

      // 最后一个分片才看结果
      if (i === chunkCount - 1) {
        if (res.code === 0) {
          const count = res.data?.imported ?? 0
          message.success(`SQL 导入成功，已执行 ${count} 条语句`)
          fetchData()
        } else {
          message.error(res.msg || '导入失败')
        }
      }
    }
  } catch (err: any) {
    message.error(err?.message || '导入请求失败')
  } finally {
    importing.value = false
    importProgress.value = 0
    target.value = ''
  }
}

const fetchData = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  loading.value = true
  try {
    const res = await getDBManagerTableListAPI({
      serverId: props.selectedServerId,
      databaseName: props.selectedDatabase,
      page: page.value,
      limit: limit.value
    })
    if (res.code === 0 && res.data) {
      tables.value = res.data.items || []
      total.value = res.data.total || 0
    } else {
      tables.value = []
      total.value = 0
    }
  } catch {
    tables.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

const fetchViews = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  loadingViews.value = true
  const t = serverType.value
  let sql = ''
  if (t === 'mysql' || t === 'mariadb') {
    sql = `SHOW FULL TABLES WHERE TABLE_TYPE = 'VIEW'`
  } else if (t === 'postgresql') {
    sql = `SELECT table_name FROM information_schema.views WHERE table_schema = 'public'`
  } else if (t === 'sqlite') {
    sql = `SELECT name FROM sqlite_master WHERE type = 'view' AND name NOT LIKE 'sqlite_%'`
  }
  if (!sql) { loadingViews.value = false; views.value = []; return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
    if (res.code === 0 && res.data && res.data.type === 'query') {
      const rows = res.data.rows || []
      if (t === 'mysql' || t === 'mariadb') {
        views.value = rows.map((r: any) => Object.values(r)[0] as string)
      } else {
        const key = Object.keys(rows[0] || {})[0]
        views.value = rows.map((r: any) => r[key] || '')
      }
    } else {
      views.value = []
    }
  } catch {
    views.value = []
  } finally {
    loadingViews.value = false
  }
}

const viewDefinitionSql = (viewName: string) => {
  const t = serverType.value
  const q = t === 'mysql' || t === 'mariadb' ? '`' : '"'
  if (t === 'mysql' || t === 'mariadb') return `SHOW CREATE VIEW ${q}${viewName}${q}`
  if (t === 'postgresql') return `SELECT pg_get_viewdef('${viewName.replace(/'/g, "''")}', true) AS definition`
  if (t === 'sqlite') return `SELECT sql FROM sqlite_master WHERE type = 'view' AND name = '${viewName.replace(/'/g, "''")}'`
  return ''
}

const openViewDefinition = async (viewName: string) => {
  const sql = viewDefinitionSql(viewName)
  if (!sql) { message.warning('当前数据库类型不支持查看视图定义'); return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql })
    if (res.code === 0 && res.data && res.data.type === 'query' && res.data.rows && res.data.rows.length > 0) {
      const row = res.data.rows[0]
      if (serverType.value === 'mysql' || serverType.value === 'mariadb') {
        viewDefinition.value = Object.values(row)[1] as string || Object.values(row)[0] as string
      } else {
        viewDefinition.value = Object.values(row)[0] as string
      }
    } else {
      viewDefinition.value = '-- 无法获取视图定义'
    }
  } catch {
    viewDefinition.value = '-- 获取视图定义失败'
  }
  viewDefinitionTitle.value = viewName
  showViewDefModal.value = true
}

const dropView = async (viewName: string) => {
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql: `DROP VIEW ${q}${viewName}${q}` })
    if (res.code === 0) {
      message.success(`视图 ${viewName} 已删除`)
      fetchViews()
    } 
  } catch {
    message.error('删除视图请求失败')
  }
}

// 函数/存储过程
const fetchRoutines = async () => {
  if (!props.selectedServerId || !props.selectedDatabase) return
  loadingRoutines.value = true
  const t = serverType.value
  let sql = ''
  if (t === 'mysql' || t === 'mariadb') {
    sql = `SELECT ROUTINE_NAME AS name, ROUTINE_TYPE AS type FROM information_schema.ROUTINES WHERE ROUTINE_SCHEMA = '${props.selectedDatabase.replace(/'/g, "''")}' ORDER BY ROUTINE_TYPE, ROUTINE_NAME`
  } else if (t === 'postgresql') {
    sql = `SELECT p.proname AS name, CASE p.prokind WHEN 'f' THEN 'FUNCTION' WHEN 'p' THEN 'PROCEDURE' ELSE 'FUNCTION' END AS type FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public' AND p.prokind IN ('f', 'p') ORDER BY p.proname`
  }
  if (!sql) { loadingRoutines.value = false; routines.value = []; return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId, databaseName: props.selectedDatabase, sql })
    if (res.code === 0 && res.data && res.data.type === 'query') {
      routines.value = (res.data.rows || []).map((r: any) => ({
        name: r.name || Object.values(r)[0] || '',
        type: r.type || (t === 'mysql' ? 'FUNCTION' : 'FUNCTION')
      }))
    } else {
      routines.value = []
    }
  } catch {
    routines.value = []
  } finally {
    loadingRoutines.value = false
  }
}

const openRoutineDefinition = async (name: string, type: string) => {
  const t = serverType.value
  const q = t === 'mysql' || t === 'mariadb' ? '`' : '"'
  let sql = ''
  if (t === 'mysql' || t === 'mariadb') {
    sql = `SHOW CREATE ${type} ${q}${name}${q}`
  } else if (t === 'postgresql') {
    sql = `SELECT pg_get_functiondef(p.oid) AS definition FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public' AND p.proname = '${name.replace(/'/g, "''")}'`
  }
  if (!sql) { message.warning('当前数据库类型不支持'); return }
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql })
    if (res.code === 0 && res.data && res.data.type === 'query' && res.data.rows && res.data.rows.length > 0) {
      const row = res.data.rows[0]
      if (t === 'mysql') {
        const val = Object.values(row)
        viewDefinition.value = (val[2] || val[1] || val[0]) as string
      } else {
        viewDefinition.value = Object.values(row)[0] as string
      }
    } else {
      viewDefinition.value = '-- 无法获取定义'
    }
  } catch {
    viewDefinition.value = '-- 获取定义失败'
  }
  viewDefinitionTitle.value = `${type} ${name}`
  showViewDefModal.value = true
}

const dropRoutine = async (name: string, type: string) => {
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  try {
    const res = await execDBManagerSqlAPI({ serverId: props.selectedServerId!, databaseName: props.selectedDatabase!, sql: `DROP ${type} IF EXISTS ${q}${name}${q}` })
    if (res.code === 0) {
      message.success(`${type} ${name} 已删除`)
      fetchRoutines()
    } 
  } catch {
    message.error('删除请求失败')
  }
}

const handleBatchTruncate = async () => {
  if (checkedKeys.value.length === 0) return
  batchLoading.value = true
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  let success = 0
  for (const name of checkedKeys.value) {
    try {
      const sql = serverType.value === 'sqlite'
        ? `DELETE FROM ${q}${name}${q}`
        : `TRUNCATE TABLE ${q}${name}${q}`
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql
      })
      if (res.code === 0) success++
    } catch { /* skip */ }
  }
  message.success(`已清空 ${success}/${checkedKeys.value.length} 个表`)
  checkedKeys.value = []
  if (success > 0) fetchData()
  batchLoading.value = false
}

// Postgres 在表被其他对象（视图、外键等）依赖时会拒绝 DROP TABLE，这里识别这个特定报错，
// 询问用户是否改用 CASCADE 连带删除依赖对象，而不是默认就带上 CASCADE 静默删掉更多东西
const isDependencyError = (msg: string) => /depend(s|ency)? on it|dependent objects/i.test(msg || '')

const dropTables = async (names: string[], cascade: boolean) => {
  const q = serverType.value === 'mysql' || serverType.value === 'mariadb' ? '`' : '"'
  let success = 0
  const dependencyFailed: string[] = []
  for (const name of names) {
    try {
      const res = await execDBManagerSqlAPI({
        serverId: props.selectedServerId!,
        databaseName: props.selectedDatabase!,
        sql: `DROP TABLE ${q}${name}${q}${cascade ? ' CASCADE' : ''}`
      })
      if (res.code === 0) {
        success++
      } else if (!cascade && serverType.value === 'postgresql' && isDependencyError(res.msg)) {
        dependencyFailed.push(name)
      }
    } catch { /* skip */ }
  }
  return { success, dependencyFailed }
}

const handleBatchDrop = async () => {
  if (checkedKeys.value.length === 0) return
  batchLoading.value = true
  const names = [...checkedKeys.value]
  const { success, dependencyFailed } = await dropTables(names, false)
  batchLoading.value = false

  if (dependencyFailed.length > 0) {
    if (success > 0) message.success(`已删除 ${success}/${names.length} 个表`)
    dialog.warning({
      title: '存在依赖对象',
      content: `以下 ${dependencyFailed.length} 个表被其他数据库对象（如视图、外键）依赖，无法直接删除：${dependencyFailed.join('、')}。是否级联删除（CASCADE）？这会连带删除所有依赖它们的对象，此操作不可恢复！`,
      positiveText: '级联删除',
      negativeText: '取消',
      onPositiveClick: async () => {
        batchLoading.value = true
        const retry = await dropTables(dependencyFailed, true)
        batchLoading.value = false
        message.success(`已级联删除 ${retry.success}/${dependencyFailed.length} 个表`)
        checkedKeys.value = []
        fetchData()
      }
    })
  } else {
    message.success(`已删除 ${success}/${names.length} 个表`)
  }
  checkedKeys.value = []
  if (success > 0) fetchData()
}

watch(() => [props.selectedServerId, props.selectedDatabase], () => {
  page.value = 1
  checkedKeys.value = []
  viewMode.value = 'tables'
  fetchData()
  fetchDbUsers()
})

watch(viewMode, (mode) => {
  if (mode === 'views') fetchViews()
  if (mode === 'routines') fetchRoutines()
})

onMounted(() => {
  fetchData()
  fetchDbUsers()
})
</script>

<template>
  <div class="flex-1 flex flex-col overflow-hidden">
    <div class="p-1.5 border-b border-slate-200 bg-white flex gap-1 text-xs">
      <n-button
        size="tiny"
        :type="viewMode === 'tables' ? 'primary' : 'default'"
        @click="viewMode = 'tables'"
      >
        <template #icon><n-icon :component="renderIcon('mdi:table-multiple')" /></template>
        表
      </n-button>
      <n-button
        size="tiny"
        :type="viewMode === 'views' ? 'primary' : 'default'"
        @click="viewMode = 'views'"
      >
        <template #icon><n-icon :component="renderIcon('mdi:eye-outline')" /></template>
        视图
      </n-button>
      <n-button
        size="tiny"
        :type="viewMode === 'routines' ? 'primary' : 'default'"
        @click="viewMode = 'routines'"
      >
        <template #icon><n-icon :component="renderIcon('mdi:code-braces')" /></template>
        函数
      </n-button>
    </div>

    <!-- 表概览 -->
    <template v-if="viewMode === 'tables'">
      <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] flex justify-between items-center text-xs">
        <div class="text-slate-700 flex items-center gap-2">
          <n-icon :component="renderIcon('mdi:table-multiple')" />
          <span class="font-semibold">表概览</span>
          <span class="text-slate-400">·</span>
          <span>{{ total }} 个表</span>
          <span v-if="checkedKeys.length > 0" class="text-blue-600 ml-2">已选 {{ checkedKeys.length }} 个</span>
        </div>
        <div class="flex gap-2 items-center">
          <template v-if="checkedKeys.length > 0">
            <n-popconfirm @positive-click="handleBatchTruncate">
              <template #trigger>
                <n-button size="tiny" type="warning" ghost :loading="batchLoading">清空选中</n-button>
              </template>
              确定要清空选中的 {{ checkedKeys.length }} 个表吗？此操作不可恢复！
            </n-popconfirm>
            <n-popconfirm @positive-click="handleBatchDrop">
              <template #trigger>
                <n-button size="tiny" type="error" ghost :loading="batchLoading">删除选中</n-button>
              </template>
              确定要删除选中的 {{ checkedKeys.length }} 个表吗？此操作不可恢复！
            </n-popconfirm>
          </template>
          <n-button size="tiny" @click="handleImportSQL" :loading="importing" type="success" ghost :disabled="importing">
            <template #icon><n-icon :component="renderIcon('mdi:file-upload-outline')" /></template>
            {{ importing ? `导入中 ${importProgress}%` : $t('database.importSql') }}
          </n-button>
          <n-button size="tiny" @click="fetchData" :loading="loading">刷新</n-button>
        </div>
      </div>
      <input
        ref="fileInputRef"
        type="file"
        accept=".sql"
        class="hidden"
        @change="onFileSelected"
      />
      <div class="flex-1 p-2 bg-white overflow-auto">
        <n-empty v-if="!loading && tables.length === 0" description="当前数据库中没有表" class="mt-10" />
        <n-data-table
          v-else
          :columns="columns as any[]"
          :data="tables"
          :loading="loading"
          :pagination="false"
          :bordered="true"
          size="small"
          class="text-xs"
          :row-key="(row: any) => row.name"
          @update:checked-row-keys="checkedKeys = $event as string[]"
        />
      </div>
      <div v-if="total > 0" class="p-2 border-t border-slate-200 bg-white flex justify-end">
        <n-pagination
          :page="page"
          :page-size="limit"
          :item-count="total"
          :page-sizes="[20, 30, 50, 100]"
          size="small"
          show-size-picker
          @update:page="page = $event; fetchData()"
          @update:page-size="limit = $event; page = 1; fetchData()"
        />
      </div>
    </template>

    <!-- 视图列表 -->
    <template v-if="viewMode === 'views'">
      <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] flex justify-between items-center text-xs">
        <div class="text-slate-700 flex items-center gap-2">
          <n-icon :component="renderIcon('mdi:eye-outline')" />
          <span class="font-semibold">视图管理</span>
          <span class="text-slate-400">·</span>
          <span>{{ views.length }} 个视图</span>
        </div>
        <div class="flex gap-2">
          <n-button size="tiny" @click="fetchViews" :loading="loadingViews">刷新</n-button>
        </div>
      </div>
      <div class="flex-1 p-2 bg-white overflow-auto">
        <n-empty v-if="!loadingViews && views.length === 0" description="当前数据库中没有视图" class="mt-10" />
        <div v-else class="flex flex-col gap-2">
          <div
            v-for="v in views"
            :key="v"
            class="flex items-center justify-between rounded border border-slate-200 px-3 py-2 hover:bg-slate-50"
          >
            <div class="flex items-center gap-2">
              <n-icon :component="renderIcon('mdi:eye-outline')" class="text-slate-400" />
              <span class="font-medium text-sm">{{ v }}</span>
            </div>
            <div class="flex gap-2">
              <n-button size="tiny" @click="openViewDefinition(v)">定义</n-button>
              <n-popconfirm @positive-click="dropView(v)">
                <template #trigger>
                  <n-button size="tiny" type="error" ghost>删除</n-button>
                </template>
                确定要删除视图 {{ v }} 吗？
              </n-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- 函数/存储过程列表 -->
    <template v-if="viewMode === 'routines'">
      <div class="p-2 border-b border-slate-200 bg-[#f8f9fa] flex justify-between items-center text-xs">
        <div class="text-slate-700 flex items-center gap-2">
          <n-icon :component="renderIcon('mdi:code-braces')" />
          <span class="font-semibold">函数与存储过程</span>
          <span class="text-slate-400">·</span>
          <span>{{ routines.length }} 个</span>
        </div>
        <div class="flex gap-2">
          <n-button size="tiny" @click="fetchRoutines" :loading="loadingRoutines">刷新</n-button>
        </div>
      </div>
      <div class="flex-1 p-2 bg-white overflow-auto">
        <n-empty v-if="!loadingRoutines && routines.length === 0" description="当前数据库中没有函数或存储过程" class="mt-10" />
        <div v-else class="flex flex-col gap-2">
          <div
            v-for="r in routines"
            :key="r.name"
            class="flex items-center justify-between rounded border border-slate-200 px-3 py-2 hover:bg-slate-50"
          >
            <div class="flex items-center gap-2">
              <n-icon :component="renderIcon(r.type === 'PROCEDURE' ? 'mdi:application-braces' : 'mdi:code-braces')" class="text-slate-400" />
              <span>
                <span class="font-medium text-sm">{{ r.name }}</span>
                <n-tag size="tiny" :type="r.type === 'PROCEDURE' ? 'info' : 'success'" class="ml-2">{{ r.type }}</n-tag>
              </span>
            </div>
            <div class="flex gap-2">
              <n-button size="tiny" @click="openRoutineDefinition(r.name, r.type)">定义</n-button>
              <n-popconfirm @positive-click="dropRoutine(r.name, r.type)">
                <template #trigger>
                  <n-button size="tiny" type="error" ghost>删除</n-button>
                </template>
                确定要删除 {{ r.type }} {{ r.name }} 吗？
              </n-popconfirm>
            </div>
          </div>
        </div>
      </div>
    </template>
  </div>

  <!-- 定义查看 Modal -->
  <n-modal
    v-model:show="showViewDefModal"
    preset="card"
    style="width: 700px"
    :title="`视图定义 - ${viewDefinitionTitle}`"
  >
    <n-input
      :value="viewDefinition"
      type="textarea"
      :rows="12"
      readonly
      class="font-mono text-xs"
    />
    <template #footer>
      <div class="flex justify-end">
        <n-button size="small" @click="showViewDefModal = false">关闭</n-button>
      </div>
    </template>
  </n-modal>
</template>
