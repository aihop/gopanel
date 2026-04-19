<script setup lang="ts">
import { ref, watch } from "vue"
import { databaseServerUpdateAPI, databaseServerGetAPI } from "@/api/modules/database"
import { useI18n } from "vue-i18n"
import emitter from "@/utils/emitter"
import { useMessage } from "naive-ui"

const { t } = useI18n()
const message = useMessage()
const show = defineModel<boolean>("show", { type: Boolean, required: true })
const id = defineModel<number>("id", { type: Number, required: true })
const updateModel = ref({
	name: "",
	host: "127.0.0.1",
	port: 3306,
	username: "",
	password: "",
	remark: "",
	id: id.value,
	type: "mysql"
})

const handleUpdate = () => {
	databaseServerUpdateAPI(updateModel.value).then(() => {
		show.value = false
		message.success(t("Modified successfully"))
		emitter.emit("database-user:refresh")
	})
}

watch(
	() => show.value,
	value => {
		if (value && id.value) {
			databaseServerGetAPI({ id: id.value }).then(({ data }: { data: any }) => {
				updateModel.value.name = data.name
				updateModel.value.host = data.host
				updateModel.value.port = data.port
				updateModel.value.username = data.username
				updateModel.value.password = data.password
				updateModel.value.remark = data.remark
				updateModel.value.id = data.id
				updateModel.value.type = data.type
			})
		}
	}
)
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="$t('database.updateServer')"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-form :model="updateModel">
      <n-form-item
        path="name"
        :label="$t('database.serverName')"
      >
        <n-input
          v-model:value="updateModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.serverName')])"
        />
      </n-form-item>
      <n-row :gutter="[0, 24]">
        <n-col :span="updateModel.type === 'sqlite' ? 24 : 15">
          <n-form-item
            path="host"
            :label="updateModel.type === 'sqlite' ? '数据库文件绝对路径' : $t('database.host')"
          >
            <n-input
              v-model:value="updateModel.host"
              :placeholder="updateModel.type === 'sqlite' ? '例如：/var/www/shoply/data.db' : $t('form.pleaseInputTo', [$t('database.serverHost')])"
            />
          </n-form-item>
        </n-col>
        <n-col
          :span="2"
          v-if="updateModel.type !== 'sqlite'"
        ></n-col>
        <n-col
          :span="7"
          v-if="updateModel.type !== 'sqlite'"
        >
          <n-form-item
            path="port"
            :label="$t('database.port')"
          >
            <n-input-number
              w-full
              :value="updateModel.port"
              @keydown.enter.prevent
              :placeholder="$t('form.pleaseInputTo', [$t('database.serverPort')])"
            />
          </n-form-item>
        </n-col>
      </n-row>
      <n-form-item
        v-if="updateModel.type !== 'sqlite'"
        path="username"
        :label="$t('database.username')"
      >
        <n-input
          :value="updateModel.username"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.username')])"
        />
      </n-form-item>
      <n-form-item
        v-if="updateModel.type !== 'sqlite'"
        path="password"
        :label="$t('database.password')"
      >
        <n-input
          :value="updateModel.password"
          type="password"
          show-password-on="click"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.password')])"
        />
      </n-form-item>
      <n-form-item
        path="remark"
        :label="$t('database.comment')"
      >
        <n-input
          :value="updateModel.remark"
          type="textarea"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.comment')])"
        />
      </n-form-item>
    </n-form>
    <n-button
      type="info"
      block
      @click="handleUpdate"
    >{{ $t("commons.button.submit") }}</n-button>
  </n-modal>
</template>
