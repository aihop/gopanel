<script setup lang="ts">
import { ref, watch } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { databaseServerCreateAPI } from "@/api/modules/database"
import emitter from "@/utils/emitter"

const message = useMessage()
const { t } = useI18n()

const show = defineModel<boolean>("show", { type: Boolean, required: true })

const importModel = ref({
	name: "",
	type: "sqlite",
	host: "", // Will hold the absolute path to the local .db file
	port: 0,
	username: "",
	password: "",
	remark: ""
})

const handleImport = () => {
	if (!importModel.value.name) {
		message.error(t("databaseManager.importSqlite.aliasRequired"))
		return
	}
	if (!importModel.value.host) {
		message.error(t("databaseManager.importSqlite.pathRequired"))
		return
	}

	databaseServerCreateAPI(importModel.value).then(() => {
		show.value = false
		message.success(t("databaseManager.importSqlite.success"))
		emitter.emit("database:refresh")
		emitter.emit("database-server:refresh")
	})
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="t('databaseManager.importSqlite.title')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @update:show="(val) => show = val"
  >
    <n-form :model="importModel">
      <n-form-item
        path="name"
        :label="t('databaseManager.importSqlite.aliasLabel')"
      >
        <n-input
          v-model:value="importModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="t('databaseManager.importSqlite.aliasPlaceholder')"
        />
      </n-form-item>

      <n-form-item
        path="host"
        :label="t('databaseManager.importSqlite.pathLabel')"
      >
        <n-input
          v-model:value="importModel.host"
          type="text"
          @keydown.enter.prevent
          :placeholder="t('databaseManager.importSqlite.pathPlaceholder')"
        />
      </n-form-item>

      <n-form-item
        path="remark"
        :label="t('databaseManager.importSqlite.commentLabel')"
      >
        <n-input
          v-model:value="importModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="t('databaseManager.importSqlite.commentPlaceholder')"
        />
      </n-form-item>
    </n-form>

    <n-alert
      type="info"
      class="mb-4"
    >
      {{ t('databaseManager.importSqlite.hint') }}
    </n-alert>

    <n-button
      type="info"
      block
      @click="handleImport"
    >{{ t('databaseManager.importSqlite.confirm') }}</n-button>
  </n-modal>
</template>