<script setup lang="ts">
import { computed, ref } from "vue"
import { useDialog, useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { cleanupCodeWorktreeResidues, getCodeWorktreeResidues } from "@/api/modules/code"
import type { CodeWorktreeResidueSummary } from "@/api/interface/codeResidues"
import Icon from "@/components/common/Icon.vue"
import { codeResidueMessages } from "../codeResidueMessages"
import CodeResidueDrawer from "./CodeResidueDrawer.vue"

const { t } = useI18n({ messages: codeResidueMessages })
const dialog = useDialog()
const message = useMessage()
const summary = ref<CodeWorktreeResidueSummary>({ residues: [], reclaimableIds: [], reclaimBytes: 0 })
const loading = ref(false)
const loadFailed = ref(false)
const cleaning = ref(false)
const showDrawer = ref(false)
const selected = ref<number[]>([])

const reclaimableCount = computed(() => summary.value.reclaimableIds.length)

// 扫描要逐个 worktree 跑 git 命令，不做轮询：残留只在交付后产生，
// 用户点开抽屉时拉一次就够，后台定时刷只会平白占住仓库。
const fetchResidues = async (silent = false) => {
	if (loading.value) return
	loading.value = true
	if (!silent) loadFailed.value = false
	try {
		const response = await getCodeWorktreeResidues()
		if (response.code !== 0) throw new Error(response.message || t("code.residueLoadFailed"))
		summary.value = response.data
		selected.value = selected.value.filter(id => summary.value.reclaimableIds.includes(id))
		loadFailed.value = false
	} catch {
		loadFailed.value = true
	} finally {
		loading.value = false
	}
}

const openDrawer = () => {
	showDrawer.value = true
	void fetchResidues()
}

const confirmClean = (sessionIds: number[]) => {
	if (sessionIds.length === 0) return
	dialog.warning({
		title: t("code.residueCleanTitle"),
		content: t("code.residueCleanConfirm", { count: sessionIds.length }),
		positiveText: t("code.residueCleanConfirmAction"),
		negativeText: t("code.cancel"),
		onPositiveClick: async () => {
			cleaning.value = true
			try {
				const response = await cleanupCodeWorktreeResidues(sessionIds)
				if (response.code !== 0) throw new Error(response.message)
				const cleaned = response.data.filter(outcome => outcome.cleaned).length
				const skipped = response.data.length - cleaned
				// 部分跳过是正常结果而不是失败：服务端在清理时会重新判定，
				// 期间变脏或重新活跃的会话会被保留下来。
				if (skipped > 0) message.warning(t("code.residueCleanPartial", { cleaned, skipped }))
				else message.success(t("code.residueCleaned", { count: cleaned }))
				selected.value = []
				await fetchResidues(true)
			} catch {
				message.error(t("code.residueCleanFailed"))
			} finally {
				cleaning.value = false
			}
		},
	})
}
</script>

<template>
  <n-button
    text
    size="tiny"
    @click="openDrawer"
  >
    <template #icon>
      <Icon
        name="mdi:broom"
        :size="14"
      />
    </template>
    {{ t("code.residueEntry") }}
    <span
      v-if="reclaimableCount > 0"
      class="ml-1 rounded-full bg-emerald-50 px-1.5 text-[10px] font-medium text-emerald-700"
    >
      {{ reclaimableCount }}
    </span>
  </n-button>

  <CodeResidueDrawer
    v-model:show="showDrawer"
    v-model:selected="selected"
    :residues="summary.residues"
    :loading="loading"
    :load-failed="loadFailed"
    :cleaning="cleaning"
    @refresh="fetchResidues()"
    @clean="confirmClean"
  />
</template>
