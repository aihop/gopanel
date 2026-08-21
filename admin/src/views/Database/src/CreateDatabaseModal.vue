<script setup lang="ts">
import { computed, ref, watch, inject, Ref } from "vue"
import { useI18n } from "vue-i18n"
import { databaseServerListAPI, databaseCreateAPI } from "@/api/modules/database"
import { useMessage } from "naive-ui"
import emitter from "@/utils/emitter"

const { t } = useI18n()
const message = useMessage()
const globalSelectedServerId = inject<Ref<number | null>>("globalSelectedServerId")

const show = defineModel<boolean>("show", { type: Boolean, required: true })
const createModel = ref({
	serverId: null,
	name: "",
	createUser: true,
	username: "",
	password: Math.random().toString(32),
	host: "localhost"
})
type MySQLHostMode = "localhost" | "%" | "specific"

const servers = ref<{ label: string; value: number; type: string }[]>([])
const hostMode = ref<MySQLHostMode>("localhost")

const hostType = [
	{ label: t("database.localhost"), value: "localhost" },
	{ label: t("database.allHost"), value: "%" },
	{ label: t("database.specificHost"), value: "specific" }
]
const selectedServerType = computed(() => servers.value.find(server => server.value === createModel.value.serverId)?.type || "")
const isMySQL = computed(() => selectedServerType.value === "mysql" || selectedServerType.value === "mariadb")

const handleHostModeChange = (value: MySQLHostMode) => {
	hostMode.value = value
	createModel.value.host = value === "specific" ? "" : value
}

const handleCreate = async () => {
	if (isMySQL.value && createModel.value.createUser && hostMode.value === "specific" && !createModel.value.host.trim()) {
		message.error(t("database.specificHostRequired"))
		return
	}
	try {
		await databaseCreateAPI(createModel.value)
		show.value = false
		message.success(t("Created successfully"))
		emitter.emit("database:refresh")
	} catch (_error) {
		message.error(t("database.databaseSaveFailed"))
	}
}

watch(
	() => show.value,
	value => {
		if (value) {
			hostMode.value = "localhost"
			createModel.value.host = "localhost"
			createModel.value.serverId = globalSelectedServerId?.value || null as any
			databaseServerListAPI({ page: 1, limit: 10000 }).then(({ data }: { data: any }) => {
				const items = Array.isArray(data) ? data : (data?.items || [])
				servers.value = []
				for (const server of items) {
					servers.value.push({
						label: server.name,
						value: server.id,
						type: server.type
					})
				}
			})
		}
	}
)

watch(
	() => selectedServerType.value,
	() => {
		createModel.value.host = isMySQL.value ? (hostMode.value === "specific" ? "" : hostMode.value) : ""
	}
)
</script>

<template>
  <n-modal
    v-model:show="show"
    :title="$t('database.createDatabase')"
    style="width: 60vw"
    size="huge"
    preset="card"
    :bordered="false"
    @close="show = false"
    :mask-closable="false"
  >
    <n-form :model="createModel">
      <n-form-item
        path="serverId"
        :label="$t('database.server')"
      >
        <n-select
          v-model:value="createModel.serverId"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseSelectTo', [$t('database.server')])"
          :options="servers"
        />
      </n-form-item>
      <n-form-item
        path="database"
        :label="$t('database.databaseName')"
      >
        <n-input
          v-model:value="createModel.name"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.databaseName')])"
        />
      </n-form-item>
      <n-form-item
        path="createUser"
        :label="$t('database.createUser')"
      >
        <n-switch v-model:value="createModel.createUser" />
      </n-form-item>
      <n-form-item
        v-if="!createModel.createUser"
        path="username"
        :label="$t('database.authorizedUser')"
      >
        <n-input
          v-model:value="createModel.username"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.authorizedUser')])"
        />
      </n-form-item>
      <n-form-item
        v-if="createModel.createUser"
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
        v-if="createModel.createUser"
        path="password"
        :label="$t('database.password')"
      >
        <n-input
          v-model:value="createModel.password"
          type="password"
          show-password-on="click"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.password')])"
        >
          <template #suffix>
            <n-button
              type="primary"
              size="tiny"
              @click="createModel.password = Math.random().toString(32)"
            >
              {{ $t("commons.button.generate") }}
            </n-button>
          </template>
        </n-input>
      </n-form-item>
      <n-form-item
        v-if="createModel.createUser && isMySQL"
        path="host-select"
        :label="$t('database.host')"
      >
        <n-select
		  :value="hostMode"
          @keydown.enter.prevent
		  @update:value="handleHostModeChange"
          :placeholder="$t('form.pleaseSelectTo', [$t('database.serverHost')])"
          :options="hostType"
        />
      </n-form-item>
      <n-form-item
        v-if="createModel.createUser && isMySQL && hostMode === 'specific'"
        path="host"
        :label="$t('database.specificHost')"
      >
        <n-input
          v-model:value="createModel.host"
          type="text"
          @keydown.enter.prevent
          :placeholder="$t('form.pleaseInputTo', [$t('database.specificHostAddress')])"
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
