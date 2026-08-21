<script setup lang="ts">
import { ref } from 'vue'
import { NModal, NInput, NSelect, NButton, useMessage, NIcon } from 'naive-ui'
import { renderIcon } from '@/utils'
import { useI18n } from 'vue-i18n'
import { createDBManagerDatabaseAPI } from '@/api/modules/database'

const props = defineProps<{
  serverId: number | null
  serverType?: string
}>()

const emit = defineEmits<{
  (e: 'success'): void
}>()

const message = useMessage()
const { t } = useI18n()
const show = defineModel<boolean>('show', { default: false })

const databaseName = ref('')
const charset = ref('utf8mb4')
const collation = ref('utf8mb4_unicode_ci')
const submitting = ref(false)

const charsetOptions = [
  { label: t('databaseManager.createDatabase.charset.utf8mb4Recommended'), value: 'utf8mb4' },
  { label: 'utf8', value: 'utf8' },
  { label: 'utf16', value: 'utf16' },
  { label: 'latin1', value: 'latin1' },
  { label: 'gbk', value: 'gbk' },
  { label: 'big5', value: 'big5' },
]

const collationOptions = [
  { label: 'utf8mb4_unicode_ci', value: 'utf8mb4_unicode_ci' },
  { label: 'utf8mb4_general_ci', value: 'utf8mb4_general_ci' },
  { label: 'utf8mb4_bin', value: 'utf8mb4_bin' },
  { label: 'utf8_general_ci', value: 'utf8_general_ci' },
  { label: 'utf8_bin', value: 'utf8_bin' },
  { label: 'latin1_swedish_ci', value: 'latin1_swedish_ci' },
  { label: 'gbk_chinese_ci', value: 'gbk_chinese_ci' },
]

const handleSubmit = async () => {
  if (!props.serverId) {
    message.warning(t('databaseManager.createDatabase.selectServerFirst'))
    return
  }
  const name = databaseName.value.trim()
  if (!name) {
    message.warning(t('databaseManager.createDatabase.nameRequired'))
    return
  }
  // 基本校验：只允许字母数字下划线
  if (!/^[a-zA-Z_][a-zA-Z0-9_\-$]*$/.test(name)) {
    message.warning(t('databaseManager.createDatabase.nameFormatInvalid'))
    return
  }

  submitting.value = true
  try {
    const payload: any = {
      serverId: props.serverId,
      databaseName: name,
    }
    if (props.serverType === 'mysql' || props.serverType === 'mariadb') {
      payload.charset = charset.value
      payload.collation = collation.value
    }
    const res = await createDBManagerDatabaseAPI(payload)
    if (res.code === 0) {
      message.success(t('databaseManager.createDatabase.success', { name }))
      show.value = false
      databaseName.value = ''
      emit('success')
    } else {
      message.error(res.message || t('databaseManager.createDatabase.failed'))
    }
  } catch (error: any) {
    void 0
  } finally {
    submitting.value = false
  }
}

const handleReset = () => {
  databaseName.value = ''
  charset.value = 'utf8mb4'
  collation.value = 'utf8mb4_unicode_ci'
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    style="width: 480px"
    :title="t('databaseManager.createDatabase.title')"
    @after-leave="handleReset"
  >
    <div class="flex flex-col gap-4 text-sm">
      <div class="flex items-center gap-2">
        <span class="w-20 text-right text-slate-600">{{ t('databaseManager.createDatabase.nameLabel') }}</span>
        <n-input
          v-model:value="databaseName"
          :placeholder="t('databaseManager.createDatabase.namePlaceholder')"
          size="small"
          class="flex-1"
          clearable
        />
      </div>

      <template v-if="serverType === 'mysql' || serverType === 'mariadb'">
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">字符集:</span>
          <n-select
            v-model:value="charset"
            :options="charsetOptions"
            size="small"
            class="flex-1"
          />
        </div>
        <div class="flex items-center gap-2">
          <span class="w-20 text-right text-slate-600">排序规则:</span>
          <n-select
            v-model:value="collation"
            :options="collationOptions"
            size="small"
            class="flex-1"
          />
        </div>
      </template>

      <div class="text-xs text-slate-400 mt-1">
        <n-icon :component="renderIcon('mdi:information-outline')" class="mr-1" />
        数据库名称只能包含字母、数字、下划线，不能以数字开头
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-2">
        <n-button size="small" @click="show = false">取消</n-button>
        <n-button
          size="small"
          type="primary"
          :loading="submitting"
          @click="handleSubmit"
        >创建</n-button>
      </div>
    </template>
  </n-modal>
</template>