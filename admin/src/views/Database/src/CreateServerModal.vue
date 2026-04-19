<script setup lang="ts">
import { watch, ref } from "vue"
import { databaseServerCreateAPI } from "@/api/modules/database"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import emitter from "@/utils/emitter"

const { t } = useI18n()
const message = useMessage()

const show = defineModel<boolean>("show", { type: Boolean, required: true })
const createModel = ref({
	name: "",
	type: "mysql",
	host: "127.0.0.1",
	port: 3306,
	username: "",
	password: "",
	remark: ""
})

const databaseType = [
	{ label: "MySQL", value: "mysql" },
	{ label: "PostgreSQL", value: "postgresql" },
	{ label: "SQLite", value: "sqlite" }
]

watch(
	() => createModel.value.type,
	value => {
		if (value === "mysql") {
			createModel.value.port = 3306
			createModel.value.host = "127.0.0.1"
		} else if (value === "postgresql") {
			createModel.value.port = 5432
			createModel.value.host = "127.0.0.1"
		} else if (value === "sqlite") {
			createModel.value.host = "/data/sqlite.db"
			createModel.value.port = 0
			createModel.value.username = ""
			createModel.value.password = ""
		}
	}
)

const handleCreate = () => {
	databaseServerCreateAPI(createModel.value).then(() => {
		show.value = false
		message.success(t("Added successfully"))
		emitter.emit("database-server:refresh")
	})
}
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$t('database.addServer')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-form :model="createModel">
      <n-form-item
        path="name"
        :label="$t('database.serverName')"
      >
        <n-input
          v-model:value="createModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.serverName')])"
        />
      </n-form-item>
      <n-form-item
        path="type"
        :label="$t('database.type')"
      >
        <n-select
          v-model:value="createModel.type"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseSelectTo', [$t('database.type')])"
          :options="databaseType"
        />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="createModel.type === 'sqlite' ? 24 : 15">
          <n-form-item
            path="host"
            :label="createModel.type === 'sqlite' ? '数据库文件绝对路径' : $t('database.host')"
          >
            <n-input
              v-model:value="createModel.host"
              type="text"
              @keydown.enter.prevent
              :placeholder="createModel.type === 'sqlite' ? '例如：/var/www/shoply/data.db' : $t('form.pleaseInputTo', [$t('database.host')])"
            />
          </n-form-item>
        </n-col>
        <n-col
          :span="2"
          v-if="createModel.type !== 'sqlite'"
        ></n-col>
        <n-col
          :span="7"
          v-if="createModel.type !== 'sqlite'"
        >
          <n-form-item
            path="port"
            :label="$t('database.port')"
          >
            <n-input-number
              w-full
              v-model:value="createModel.port"
              @keydown.enter.prevent
              :placeholder="$t('form.pleaseInputTo', [$t('Port')])"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item
        v-if="createModel.type !== 'sqlite'"
        path="username"
        :label="$t('database.username')"
      >
        <n-input
          v-model:value="createModel.username"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.username')])"
        />
      </n-form-item>
      <n-form-item
        v-if="createModel.type !== 'sqlite'"
        path="password"
        :label="$t('database.password')"
      >
        <n-input
          v-model:value="createModel.password"
          type="password"
          show-password-on="click"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.password')])"
        />
      </n-form-item>
      <n-form-item
        path="remark"
        :label="$t('database.remark')"
      >
        <n-input
          v-model:value="createModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.remark')])"
        />
      </n-form-item>
    </n-form>
    <n-button
      type="info"
      block
      @click="handleCreate"
    >{{ $t("commons.button.submit") }}</n-button>
  </n-modal>
</template>
