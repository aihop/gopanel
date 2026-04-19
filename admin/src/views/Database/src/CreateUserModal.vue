<script setup lang="ts">
import { useI18n } from "vue-i18n"
import { databaseListAPI, databaseServerListAPI, databaseUserCreateAPI } from "@/api/modules/database"
import { ref, watch, inject, Ref } from "vue"
import { useMessage } from "naive-ui"
import emitter from "@/utils/emitter"

const { t } = useI18n()
const message = useMessage()
const globalSelectedServerId = inject<Ref<number | null>>("globalSelectedServerId")

const show = defineModel<boolean>("show", { type: Boolean, required: true })
const createModel = ref({
	serverId: null,
	username: "",
	password: Math.random().toString(32),
	host: "localhost",
	privileges: [] as string[],
	remark: ""
})

const servers = ref<{ label: string; value: string }[]>([])
const privilegeOptions = ref<{ label: string; value: string }[]>([])

const hostType = [
	{ label: t("database.localhost"), value: "localhost" },
	{ label: t("database.allHost"), value: "%" },
	{ label: t("database.specificHost"), value: "" }
]

const handleCreate = () => {
	databaseUserCreateAPI(createModel.value).then(() => {
		show.value = false
		message.success(t("Created successfully"))
		emitter.emit("database-user:refresh")
	})
}

const loadPrivilegeOptions = async (currentServerId: number | null) => {
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
}

watch(
	() => show.value,
	value => {
		if (value) {
			createModel.value.serverId = globalSelectedServerId?.value || null as any
			createModel.value.privileges = []
			void loadPrivilegeOptions(createModel.value.serverId)
			databaseServerListAPI({}).then(res => {
				servers.value = []
				for (const server of res.data as any[]) {
					servers.value.push({
						label: server.name,
						value: server.id
					})
				}
			})
		}
	}
)

watch(
	() => createModel.value.serverId,
	value => {
		void loadPrivilegeOptions(value as number | null)
	}
)
</script>

<template>
	<n-modal
		v-model:show="show"
		preset="card"
		:title="$t('database.createUser')"
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
			<n-form :model="createModel">
				<n-form-item path="serverId" :label="$t('database.server')">
					<n-select
						v-model:value="createModel.serverId"
						@keydown.enter.prevent
						:placeholder="$t('form.pleaseSelectTo', [$t('database.serverHost')])"
						:options="servers"
					/>
				</n-form-item>
				<n-form-item path="username" :label="$t('database.username')">
					<n-input
						v-model:value="createModel.username"
						type="text"
						@keydown.enter.prevent
						:placeholder="$t('form.pleaseInputTo', [$t('database.username')])"
					/>
				</n-form-item>
				<n-form-item path="password" :label="$t('database.password')">
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
				<n-form-item path="host-select" :label="$t('database.HostMysql')">
					<n-select
						v-model:value="createModel.host"
						@keydown.enter.prevent
						:placeholder="$t('form.pleaseSelectTo', [$t('database.host')])"
						:options="hostType"
					/>
				</n-form-item>
				<n-form-item v-if="createModel.host === ''" path="host" :label="$t('database.specificHost')">
					<n-input
						v-model:value="createModel.host"
						type="text"
						@keydown.enter.prevent
						:placeholder="$t('form.pleaseInputTo', [$t('database.specificHostAddress')])"
					/>
				</n-form-item>
				<n-form-item path="privileges" :label="$t('database.privileges')">
					<n-select
						v-model:value="createModel.privileges"
						multiple
						filterable
						tag
						clearable
						:options="privilegeOptions"
						:placeholder="$t('form.pleaseSelectTo', [$t('database.privileges')])"
					/>
				</n-form-item>
				<n-form-item path="remark" :label="$t('database.comment')">
					<n-input
						v-model:value="createModel.remark"
						type="textarea"
						@keydown.enter.prevent
						:placeholder="$t('form.pleaseInputTo', [$t('database.comment')])"
					/>
				</n-form-item>
			</n-form>
			<n-button type="info" block @click="handleCreate">{{ $t("commons.button.submit") }}</n-button>
		</n-flex>
	</n-modal>
</template>
