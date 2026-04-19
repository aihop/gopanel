<script setup lang="ts">
import { ref, watch } from "vue"
import { useMessage } from "naive-ui"
import { databaseServerCreateAPI } from "@/api/modules/database"
import emitter from "@/utils/emitter"

const message = useMessage()

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
		message.error("请输入数据库别名")
		return
	}
	if (!importModel.value.host) {
		message.error("请指定 SQLite 数据库文件路径")
		return
	}

	databaseServerCreateAPI(importModel.value).then(() => {
		show.value = false
		message.success("SQLite 数据库导入成功")
		emitter.emit("database:refresh")
		emitter.emit("database-server:refresh")
	})
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    title="导入本地 SQLite 数据库"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @update:show="(val) => show = val"
  >
    <n-form :model="importModel">
      <n-form-item
        path="name"
        label="数据库别名"
      >
        <n-input
          v-model:value="importModel.name"
          type="text"
          @keydown.enter.prevent
          placeholder="例如：本地业务数据库"
        />
      </n-form-item>

      <n-form-item
        path="host"
        label="服务器本地 .db 文件绝对路径"
      >
        <n-input
          v-model:value="importModel.host"
          type="text"
          @keydown.enter.prevent
          placeholder="例如：/var/www/shoply/data.db"
        />
      </n-form-item>

      <n-form-item
        path="remark"
        label="备注说明"
      >
        <n-input
          v-model:value="importModel.remark"
          type="textarea"
          @keydown.enter.prevent
          placeholder="选填"
        />
      </n-form-item>
    </n-form>

    <n-alert
      type="info"
      class="mb-4"
    >
      注意：这里的路径指的是 GoPanel 所在服务器上的本地绝对路径。导入后即可直接在工作台执行 SQL 管理数据。
    </n-alert>

    <n-button
      type="info"
      block
      @click="handleImport"
    >确认导入</n-button>
  </n-modal>
</template>
