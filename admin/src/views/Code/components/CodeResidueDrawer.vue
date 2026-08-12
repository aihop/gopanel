<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import type { CodeResidueState, CodeWorktreeResidue } from "@/api/interface/codeResidues"
import Icon from "@/components/common/Icon.vue"
import { computeSizeFromByte } from "@/utils/util"
import { codeResidueMessages } from "../codeResidueMessages"

const props = defineProps<{
	show: boolean
	residues: CodeWorktreeResidue[]
	loading: boolean
	loadFailed: boolean
	cleaning: boolean
	selected: number[]
}>()
const emit = defineEmits<{
	"update:show": [show: boolean]
	"update:selected": [selected: number[]]
	refresh: []
	clean: [sessionIds: number[]]
}>()
const { t } = useI18n({ messages: codeResidueMessages })

// 只有 safe 和 orphan 能选：其余两种是服务端明确拒绝清理的，
// 让它们可勾选只会制造「点了没反应」的困惑。
const reclaimable = computed(() => props.residues.filter(residue => isReclaimable(residue.state)))
const isReclaimable = (state: CodeResidueState) => state === "safe" || state === "orphan"

const stateTone: Record<CodeResidueState, string> = {
	safe: "bg-emerald-50 text-emerald-700",
	orphan: "bg-slate-100 text-slate-600",
	dirty: "bg-amber-50 text-amber-700",
	unmerged: "bg-rose-50 text-rose-700",
	active: "bg-blue-50 text-blue-700",
}

const toggle = (sessionId: number, checked: boolean) => {
	const next = checked
		? [...new Set([...props.selected, sessionId])]
		: props.selected.filter(id => id !== sessionId)
	emit("update:selected", next)
}
</script>

<template>
  <n-drawer
    :show="show"
    placement="right"
    style="width: min(680px, 100vw)"
    @update:show="emit('update:show', $event)"
  >
    <n-drawer-content
      :title="t('code.residueTitle')"
      closable
      body-content-style="padding: 16px;"
    >
      <div class="mb-4 flex items-start justify-between gap-3">
        <p class="text-xs text-slate-400">
          {{ t("code.residueHint") }}
        </p>
        <n-button
          quaternary
          circle
          size="small"
          :loading="loading"
          :aria-label="t('code.residueRefresh')"
          @click="emit('refresh')"
        >
          <template #icon>
            <Icon
              name="mdi:refresh"
              :size="16"
            />
          </template>
        </n-button>
      </div>

      <div
        v-if="loading && residues.length === 0"
        class="flex min-h-[240px] items-center justify-center"
      >
        <n-spin size="small" />
      </div>
      <div
        v-else-if="loadFailed && residues.length === 0"
        class="flex min-h-[240px] flex-col items-center justify-center gap-2 text-xs text-red-500"
      >
        <span>{{ t("code.residueLoadFailed") }}</span>
        <n-button
          text
          type="primary"
          size="tiny"
          @click="emit('refresh')"
        >
          {{ t("code.retry") }}
        </n-button>
      </div>
      <n-empty
        v-else-if="residues.length === 0"
        :description="t('code.residueEmpty')"
      />
      <div
        v-else
        class="space-y-3"
      >
        <section
          v-for="residue in residues"
          :key="residue.sessionId"
          class="rounded-xl border border-slate-200/80 p-3"
          :class="{ 'opacity-70': !isReclaimable(residue.state) }"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-start gap-2">
              <n-checkbox
                :checked="selected.includes(residue.sessionId)"
                :disabled="!isReclaimable(residue.state) || cleaning"
                class="mt-0.5"
                @update:checked="toggle(residue.sessionId, $event)"
              />
              <div class="min-w-0">
                <div class="flex items-center gap-1.5 text-sm font-semibold text-slate-700">
                  <span class="truncate">{{ residue.sessionTitle || t("code.residueSession", { id: residue.sessionId }) }}</span>
                  <span class="shrink-0 font-mono text-[10px] text-slate-400">#{{ residue.sessionId }}</span>
                </div>
                <div class="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11px] text-slate-500">
                  <span>{{ t("code.residueDirectories", { count: residue.directories.length }) }}</span>
                  <span class="font-medium text-slate-600">{{ computeSizeFromByte(residue.diskBytes) }}</span>
                </div>
                <div
                  v-if="residue.branches?.length"
                  class="mt-1 flex min-w-0 items-start gap-1.5 text-[10px] text-slate-400"
                >
                  <span class="shrink-0">{{ t("code.residueBranches") }}</span>
                  <span class="min-w-0 break-all font-mono text-slate-500">{{ residue.branches.join(", ") }}</span>
                </div>
                <p
                  v-if="residue.reason"
                  class="mt-1 text-[11px] text-slate-500"
                >
                  {{ residue.reason }}
                </p>
              </div>
            </div>
            <span
              class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium"
              :class="stateTone[residue.state]"
            >
              {{ t(`code.residueState_${residue.state}`) }}
            </span>
          </div>
        </section>
      </div>

      <template #footer>
        <div class="flex w-full items-center justify-between gap-3">
          <span class="text-[11px] text-slate-400">
            {{
              reclaimable.length
                ? t("code.residueReclaimable", {
                  count: reclaimable.length,
                  size: computeSizeFromByte(reclaimable.reduce((total, residue) => total + residue.diskBytes, 0))
                })
                : t("code.residueNothingReclaimable")
            }}
          </span>
          <div class="flex shrink-0 items-center gap-2">
            <n-button
              size="small"
              :disabled="selected.length === 0 || cleaning"
              :loading="cleaning"
              @click="emit('clean', selected)"
            >
              {{ t("code.residueCleanSelected") }}
            </n-button>
            <n-button
              size="small"
              type="primary"
              :disabled="reclaimable.length === 0 || cleaning"
              :loading="cleaning"
              @click="emit('clean', reclaimable.map(residue => residue.sessionId))"
            >
              {{ t("code.residueCleanAll") }}
            </n-button>
          </div>
        </div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>
