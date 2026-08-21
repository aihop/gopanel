<script setup lang="ts">
import { databaseListAPI, databaseUserGetAPI, databaseUserUpdateAPI } from "@/api/modules/database"
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"
import emitter from "@/utils/emitter"
import { useMessage } from "naive-ui"

const { t } = useI18n()
const mes = useMessage()

const show = defineModel<boolean>("show", { type: Boolean, required: true })
const id = defineModel<number>("id", { type: Number, required: true })
const serverId = defineModel<number>("serverId", { type: Number, default: 0 })
const username = defineModel<string>("username", { type: String, default: "" })
const host = defineModel<string>("host", { type: String, default: "" })
const serverName = ref("")
const serverType = ref("")
type MySQLHostMode = "localhost" | "%" | "specific"
const updateModel = ref({
	serverId: 0,
	username: "",
	host: "",
	password: "",
	privileges: [] as string[],
	remark: "",
	id: id.value
})
const privilegeOptions = ref<{ label: string; value: string }[]>([])
const hostMode = ref<MySQLHostMode>("localhost")

const isDetachedUser = computed(() => !id.value && !!serverId.value && !!username.value)
const userDisplayName = computed(() => {
	const u = updateModel.value.username || username.value || ""
	if (!u) return ""
	const h = updateModel.value.host || host.value || ""
	const identity = h ? `${u}@${h}` : u
	const sn = serverName.value ? `（${serverName.value}）` : ""
	return `${identity}${sn}`
})
const modalTitle = computed(() => {
	const base = t("database.updateUser")
	if (!userDisplayName.value) return base
	return `${base}：${userDisplayName.value}`
})
const isMySQL = computed(() => serverType.value === "mysql" || serverType.value === "mariadb")
const hostType = [
	{ label: t("database.localhost"), value: "localhost" },
	{ label: t("database.allHost"), value: "%" },
	{ label: t("database.specificHost"), value: "specific" }
]

const syncHostMode = (value: string) => {
	if (value === "localhost" || value === "%") {
		hostMode.value = value
		return
	}
	hostMode.value = "specific"
}

const handleHostModeChange = (value: MySQLHostMode) => {
	hostMode.value = value
	updateModel.value.host = value === "specific" ? "" : value
}

const privilegeMode = ref<"all" | "common" | "custom">("common")

const allPrivilegeValues = computed(() => privilegeOptions.value.map(item => item.value))

const commonPrivilegeValues = computed(() => {
	const systemDbNames = new Set([
		"information_schema",
		"mysql",
		"performance_schema",
		"sys",
		"postgres",
		"template0",
		"template1"
	])
	return privilegeOptions.value
		.map(item => item.value)
		.filter(name => !!name && !systemDbNames.has(String(name).toLowerCase()))
})

const syncPrivilegeModeFromSelection = () => {
	const all = allPrivilegeValues.value
	if (!all.length) return

	const selected = new Set((updateModel.value.privileges || []).filter(Boolean))
	const isAll = all.every(v => selected.has(v))
	if (isAll) {
		privilegeMode.value = "all"
		return
	}

	const common = commonPrivilegeValues.value
	const isCommon =
		common.length > 0 &&
		common.every(v => selected.has(v)) &&
		Array.from(selected).every(v => common.includes(v))
	if (isCommon) {
		privilegeMode.value = "common"
		return
	}

	privilegeMode.value = "custom"
}

const applyPrivilegeMode = (mode: "all" | "common" | "custom") => {
	privilegeMode.value = mode
	if (mode === "all") {
		updateModel.value.privileges = [...allPrivilegeValues.value]
		return
	}
	if (mode === "common") {
		updateModel.value.privileges = [...commonPrivilegeValues.value]
		return
	}
}

const handlePrivilegeModeChange = (val: string) => {
	if (val === "all" || val === "common" || val === "custom") {
		applyPrivilegeMode(val)
	}
}

const loadPrivilegeOptions = async (currentServerId: number) => {
	if (!currentServerId) {
		privilegeOptions.value = []
		return
	}
	const res: any = await databaseListAPI({
		page: 1,
		limit: 500,
		wheres: [
			{
				field: "server_id",
				rule: "eq",
				val: currentServerId.toString()
			}
		]
	})
	const items = Array.isArray(res.data?.items) ? res.data.items : []
	privilegeOptions.value = items.map((item: any) => ({
		label: item.name,
		value: item.name
	}))
	syncPrivilegeModeFromSelection()
}

const handleUpdate = async () => {
	if (isMySQL.value && hostMode.value === "specific" && !updateModel.value.host.trim()) {
		mes.error(t("database.specificHostRequired"))
		return
	}
	try {
		await databaseUserUpdateAPI(updateModel.value)
		show.value = false
		mes.success(t("Modified successfully"))
		emitter.emit("database-user:refresh")
	} catch (_error) {
		mes.error(t("database.userSaveFailed"))
	}
}

watch(
	() => show.value,
	value => {
		if (!value) return

		updateModel.value.id = id.value
		updateModel.value.serverId = serverId.value || 0
		updateModel.value.username = username.value || ""
		updateModel.value.host = host.value || ""
		updateModel.value.password = ""
		updateModel.value.privileges = []
		updateModel.value.remark = ""
		serverName.value = ""
		serverType.value = ""
		void loadPrivilegeOptions(updateModel.value.serverId)

		const getParams: any = id.value
			? { id: id.value }
			: {
					id: 0,
					serverId: updateModel.value.serverId,
					username: updateModel.value.username,
					host: updateModel.value.host
				}

		if (id.value || isDetachedUser.value) {
			databaseUserGetAPI(getParams).then(({ data }: { data: any }) => {
				updateModel.value.serverId = data.serverId || serverId.value || 0
				updateModel.value.username = data.username || username.value || ""
				updateModel.value.host = data.host || host.value || ""
				serverName.value = data.server?.name || ""
				serverType.value = data.server?.type || ""
				if (!isMySQL.value) updateModel.value.host = ""
				else syncHostMode(updateModel.value.host)
				updateModel.value.password = ""
				updateModel.value.privileges = data.privileges
				updateModel.value.remark = data.remark
				void loadPrivilegeOptions(updateModel.value.serverId)
			})
		} else {
			privilegeMode.value = "common"
		}
	}
)
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    :title="modalTitle"
    style="width: 60vw"
    size="huge"
    :bordered="false"
    :segmented="false"
    @close="show = false"
  >
    <n-flex vertical>
      <n-alert type="info">
        {{ $t("database.createUserTips") }}
      </n-alert>
      <n-alert
        v-if="isDetachedUser"
        type="warning"
      >
        {{ $t("database.detachedUserTips") }}
      </n-alert>
      <n-form :model="updateModel">
        <n-form-item
          v-if="isMySQL"
          path="host-select"
          :label="$t('database.HostMysql')"
        >
          <n-select
			:value="hostMode"
            :options="hostType"
			@update:value="handleHostModeChange"
            :placeholder="$t('form.pleaseSelectTo', [$t('database.host')])"
          />
        </n-form-item>
        <n-form-item
          v-if="isMySQL && hostMode === 'specific'"
          path="host"
          :label="$t('database.specificHost')"
        >
          <n-input
            v-model:value="updateModel.host"
            type="text"
            @keydown.enter.prevent
            :placeholder="$t('form.pleaseInputTo', [$t('database.specificHostAddress')])"
          />
        </n-form-item>
        <n-form-item
          path="password"
          :label="$t('database.password')"
        >
          <n-input
            v-model:value="updateModel.password"
            type="password"
            show-password-on="click"
            @keydown.enter.prevent
            :placeholder="$t('form.pleaseInputTo', [$t('database.password')])"
          />
        </n-form-item>
        <n-form-item
          path="privileges"
          :label="$t('database.privileges')"
        >
          <n-flex
            vertical
            class="w-full"
          >
            <n-radio-group
              :value="privilegeMode"
              @update:value="handlePrivilegeModeChange"
            >
              <n-radio value="all">{{ $t("database.allPrivileges") }}</n-radio>
              <n-radio value="common">{{ $t("database.commonPrivileges") }}</n-radio>
              <n-radio value="custom">{{ $t("database.customPrivileges") }}</n-radio>
            </n-radio-group>
            <n-select
              v-model:value="updateModel.privileges"
              multiple
              filterable
              tag
              clearable
              :disabled="privilegeMode !== 'custom'"
              :options="privilegeOptions"
              :placeholder="
                privilegeMode === 'custom'
                  ? $t('form.pleaseSelectTo', [$t('database.privileges')])
                  : privilegeMode === 'all'
                    ? $t('database.allDatabasesSelected')
                    : $t('database.commonDatabasesSelected')
              "
            />
          </n-flex>
        </n-form-item>
        <n-form-item
          path="remark"
          :label="$t('database.comment')"
        >
          <n-input
            v-model:value="updateModel.remark"
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
    </n-flex>
  </n-modal>
</template>
