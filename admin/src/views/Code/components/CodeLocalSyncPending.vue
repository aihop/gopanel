<script setup lang="ts">
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import { codeGitReviewMessages } from "../codeGitReviewMessages"

/**
 * 只声明用得到的字段，不绑定具体接口类型。
 * 推送状态（CodeDeliveryPushRepository）和交付结果（CodeRepositoryDeliveryResult）
 * 是两个不同的结构，但都带着这几个本地同步字段，用结构化类型两边都能传。
 */
interface LocalSyncRepository {
	repositoryName: string
	commit?: string
	localSynced: boolean
	localSyncError?: string
	localSyncCommand?: string
}

const props = defineProps<{ repositories: LocalSyncRepository[] }>()
const { t } = useI18n({ messages: codeGitReviewMessages })
const message = useMessage()

/**
 * 只列「交付提交产出了、但没能快进到本地主仓」的仓库。
 *
 * localSynced 由后端的 SourceAppliedAt 推出，而它只在 merge-base --is-ancestor
 * 校验通过后才写入 —— 这是「确实进了本地目标分支」的唯一可信信号。
 */
/**
 * 产出了交付提交、但没进本地主仓的仓库。
 *
 * 不能只看 localSyncError：重试路径下已标 completed 的仓库会跳过同步，
 * 结果 localSynced 为 false 而 localSyncError 为空，那样这条提示就漏了 ——
 * 而按钮那边（useCodeDelivery 的 pendingLocalSync）是按 facts.local 判的，
 * 两边口径必须一致，否则会出现「按钮说待同步、下面却什么都不列」。
 */
const pending = computed(() =>
	props.repositories.filter(item => !item.localSynced && (item.localSyncError || item.commit)),
)

const commands = computed(() =>
	pending.value
		.map(item => item.localSyncCommand)
		.filter(Boolean)
		.join("\n"),
)

async function copyCommands() {
	if (!commands.value) return
	try {
		await navigator.clipboard.writeText(commands.value)
		message.success(t("code.gitLocalSyncCopied"))
	} catch {
		message.error(t("code.gitPushStatusFailed"))
	}
}
</script>

<template>
  <!--
		本地主仓未同步是降级提示，不是交付失败：用 info 而非 error，并直接给出可执行命令。
		单仓（CodeDeliveryPush）和多仓（CodeGitReview 隔离分支）共用这一个组件 ——
		之前这段只在单仓路径上，多仓恰恰是问题高发处却看不到任何提示。
	-->
  <n-alert
    v-if="pending.length"
    type="info"
    :show-icon="false"
    class="mt-2"
  >
    <div class="space-y-2">
      <p class="text-xs font-medium text-slate-700">
        {{ t("code.gitLocalSyncPending", { count: pending.length }) }}
      </p>
      <p class="text-[11px] leading-5 text-slate-500">
        {{ t("code.gitLocalSyncHint") }}
      </p>
      <ul class="space-y-1">
        <li
          v-for="item in pending"
          :key="item.repositoryName"
          class="text-[11px] leading-5 text-slate-500"
        >
          <span class="font-medium text-slate-600">{{ item.repositoryName }}</span>
          <span class="mx-1 text-slate-300">·</span>
          <span>{{ item.localSyncError || t("code.gitLocalSyncUnknownReason") }}</span>
        </li>
      </ul>
      <pre
        v-if="commands"
        class="overflow-x-auto whitespace-pre rounded-lg bg-slate-900 p-2 font-mono text-[11px] leading-5 text-slate-100"
      >{{ commands }}</pre>
      <n-button
        v-if="commands"
        size="tiny"
        secondary
        @click="copyCommands"
      >
        {{ t("code.gitLocalSyncCopy") }}
      </n-button>
    </div>
  </n-alert>
</template>
