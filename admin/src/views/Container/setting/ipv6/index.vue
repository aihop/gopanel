<template>
  <div>
    <n-drawer
      v-model:show="drawerVisible"
      :destroy-on-close="true"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      @close="handleClose"
      size="30%"
    >
      <n-drawer-content>
        <template #header>
          <DrawerHeader
            header="IPv6"
            :back="handleClose"
          />
        </template>
        <n-alert
          class="common-prompt"
          :closable="false"
          type="warning"
        >
          <template #default>
            <span class="input-help">
              {{ $t("container.ipv6Helper") }}
              <n-link
                style="font-size: 12px; margin-left: 5px"
                icon="Position"
                @click="toDoc()"
                type="primary"
              >
                {{ $t("firewall.quickJump") }}
              </n-link>
            </span>
          </template>
        </n-alert>

        <n-form
          :model="form"
          ref="formRef"
          :rules="rules"
          v-loading="loading"
          label-position="top"
        >
          <n-row
            type="flex"
            justify="center"
          >
            <n-col :span="22">
              <n-form-item
                prop="fixedCidrV6"
                :label="$t('container.subnet')"
              >
                <n-input v-model:value="form.fixedCidrV6" />
                <span class="input-help">{{ $t("container.ipv6CidrHelper") }}</span>
              </n-form-item>
              <n-form-item>
                <n-checkbox
                  v-model="showMore"
                  :label="$t('app.advanced')"
                />
              </n-form-item>
              <div v-if="showMore">
                <n-form-item
                  prop="ip6Tables"
                  label="ip6tables"
                >
                  <n-switch v-model="form.ip6Tables"></n-switch>
                  <span class="input-help">{{ $t("container.ipv6TablesHelper") }}</span>
                </n-form-item>
                <n-form-item
                  prop="experimental"
                  label="experimental"
                >
                  <n-switch v-model="form.experimental"></n-switch>
                  <span class="input-help">{{ $t("container.experimentalHelper") }}</span>
                </n-form-item>
              </div>
            </n-col>
          </n-row>
        </n-form>
        <template #footer>
          <span class="dialog-footer">
            <n-button @click="handleClose">{{ $t("commons.button.cancel") }}</n-button>
            <n-button
              :disabled="loading"
              type="primary"
              @click="onSave(formRef)"
            >
              {{ $t("commons.button.confirm") }}
            </n-button>
          </span>
        </template>
      </n-drawer-content>
    </n-drawer>

    <ConfirmDialog
      ref="confirmDialogRef"
      @confirm="onSubmitSave"
    ></ConfirmDialog>
  </div>
</template>
<script lang="ts" setup>
import { reactive, ref } from "vue"
import { MsgSuccess } from "@/utils/message"
import { updateIpv6Option } from "@/api/modules/container"
import DrawerHeader from "@/components/DrawerHeader.vue"
import { checkIpV6 } from "@/utils/util"
import GlobalStore from "@/store/modules/global"
import { useI18n } from "vue-i18n"
const { t } = useI18n()

const globalStore = GlobalStore()
const loading = ref()
const drawerVisible = ref()
const confirmDialogRef = ref()
const formRef = ref()
const showMore = ref(true)

interface DialogProps {
	fixedCidrV6: string
	ip6Tables: boolean
	experimental: boolean
}

const form = reactive({
	fixedCidrV6: "",
	ip6Tables: false,
	experimental: false
})
const rules = reactive({
	fixedCidrV6: [{ validator: checkFixedCidrV6, trigger: "blur", required: true }]
})

function checkFixedCidrV6(rule: any, value: any, callback: any) {
	if (!form.fixedCidrV6 || form.fixedCidrV6.indexOf("/") === -1) {
		return callback(new Error(t("commons.rule.formatErr")))
	}
	if (checkIpV6(form.fixedCidrV6.split("/")[0])) {
		return callback(new Error(t("commons.rule.formatErr")))
	}
	const reg = /^(?:[1-9]|[1-9][0-9]|1[0-1][0-9]|12[0-8])$/
	if (!reg.test(form.fixedCidrV6.split("/")[1])) {
		return callback(new Error(t("commons.rule.formatErr")))
	}
	callback()
}

const toDoc = () => {
	window.open(globalStore.docsUrl + "/user_manual/container/setting/", "_blank", "noopener,noreferrer")
}

const emit = defineEmits<{ (e: "search"): void }>()

const acceptParams = (params: DialogProps): void => {
	form.fixedCidrV6 = params.fixedCidrV6
	form.ip6Tables = params.ip6Tables
	form.experimental = params.experimental
	drawerVisible.value = true
}

const onSave = async (formEl: any | undefined) => {
	if (!formEl) return
	formEl.validate(async valid => {
		if (!valid) return
		let params = {
			header: t("database.confChange"),
			operationInfo: t("database.restartNowHelper"),
			submitInputInfo: t("database.restartNow")
		}
		confirmDialogRef.value!.acceptParams(params)
	})
}

const onSubmitSave = async () => {
	loading.value = true
	await updateIpv6Option(form.fixedCidrV6, form.ip6Tables, form.experimental)
		.then(() => {
			loading.value = false
			drawerVisible.value = false
			emit("search")
			MsgSuccess(t("commons.msg.operationSuccess"))
		})
		.catch(() => {
			loading.value = false
		})
}

const handleClose = () => {
	emit("search")
	drawerVisible.value = false
}

defineExpose({
	acceptParams
})
</script>
