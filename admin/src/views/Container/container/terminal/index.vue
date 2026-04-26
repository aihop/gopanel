<template>
  <n-drawer
    v-model:show="terminalVisible"
    width="50%"
    @update:show="val => !val && handleClose()"
  >
    <n-drawer-content
      :title="$t('container.containerTerminal')"
      closable
    >
      <template #header>
        <DrawerHeader
          :header="$t('container.containerTerminal')"
          :resource="title"
          :back="handleClose"
        />
      </template>
      <n-form
        ref="formRef"
        :model="form"
        label-placement="top"
        :rules="rules"
      >
        <div
          v-if="runtimeSummary"
          class="mb-4 rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600"
        >
          {{ runtimeSummary }}
        </div>
        <n-form-item
          :label="$t('commons.table.user')"
          path="user"
        >
          <n-input
            v-model:value="form.user"
            placeholder="root"
            clearable
          />
        </n-form-item>
        <n-form-item
          :label="$t('container.command')"
          path="command"
        >
          <n-input-group class="flex items-center">
            <n-checkbox
              v-model:checked="form.isCustom"
              class="w-[100px] text-gray-500"
              @update:checked="onChangeCommand"
            >
              {{ $t("container.custom") }}
            </n-checkbox>
            <n-input
              v-if="form.isCustom"
              v-model:value="form.command"
              clearable
              style="width: 100%"
            />
            <n-select
              v-else
              v-model:value="form.command"
              style="width: 100%"
              filterable
              clearable
              :options="[
								{ label: '/bin/ash', value: '/bin/ash' },
								{ label: '/bin/bash', value: '/bin/bash' },
								{ label: '/bin/sh', value: '/bin/sh' }
							]"
            />
          </n-input-group>
        </n-form-item>
      </n-form>
      <n-button
        v-if="!terminalOpen"
        type="primary"
        style="margin-top: 10px"
        @click="initTerm"
      >
        {{ $t("commons.button.conn") }}
      </n-button>
      <n-button
        v-else
        style="margin-top: 10px"
        @click="onClose()"
      >
        {{ $t("commons.button.disconnect") }}
      </n-button>
      <Terminal
        v-if="terminalOpen"
        ref="terminalRef"
        style="height: calc(100vh - 312px); margin-top: 18px"
      ></Terminal>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts" setup>
import type { FormInst } from "naive-ui"
import DrawerHeader from "@/components/DrawerHeader.vue"
import Terminal from "@/components/Terminal.vue"
import { nextTick, reactive, ref } from "vue"
import { useI18n } from "vue-i18n"

const { t } = useI18n()
const title = ref()
const terminalVisible = ref(false)
const terminalOpen = ref(false)
const runtimeSummary = ref("")
const form = reactive({
	isCustom: false,
	command: "",
	user: "",
	containerID: "",
	runtimeHost: ""
})
const formRef = ref<FormInst>()
const terminalRef = ref<InstanceType<typeof Terminal> | null>(null)

const rules = {
	command: {
		required: true,
		message: t("commons.rule.requiredInput"),
		trigger: ["input", "blur"]
	}
}

interface DialogProps {
	containerID: string
	container: string
	runtimeHost?: string
	runtimeSummary?: string
}
async function acceptParams(params: DialogProps): Promise<void> {
	terminalVisible.value = true
	form.containerID = params.containerID
	form.runtimeHost = params.runtimeHost || ""
	runtimeSummary.value = params.runtimeSummary || ""
	title.value = params.container
	form.isCustom = false
	form.user = ""
	form.command = "/bin/sh"
	terminalOpen.value = false
}

async function onChangeCommand() {
	form.command = ""
}

async function initTerm() {
	try {
		await formRef.value?.validate()
		terminalOpen.value = true
		await nextTick()
		terminalRef.value!.acceptParams({
			endpoint: "/container/exec",
			args: `source=container&containerid=${form.containerID}&user=${form.user}&command=${form.command}&runtimeHost=${encodeURIComponent(form.runtimeHost || "")}`,
			error: "",
			initCmd: ""
		})
	} catch (errors) {
		console.log("validation failed", errors)
	}
}

function onClose() {
	terminalRef.value?.onClose()
	terminalOpen.value = false
}

function handleClose() {
	onClose()
	terminalVisible.value = false
	runtimeSummary.value = ""
}

defineExpose({
	acceptParams
})
</script>
