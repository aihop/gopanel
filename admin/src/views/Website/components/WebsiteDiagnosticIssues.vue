<template>
	<div class="space-y-4">
		<div class="flex items-center justify-between gap-3">
			<n-select v-model:value="status" :options="statusOptions" style="width: 180px" @update:value="load" />
			<n-button :loading="loading" @click="load">{{ t("websiteDiagnostic.refresh") }}</n-button>
		</div>
		<n-alert v-if="error" type="error" :show-icon="true">{{ error }}</n-alert>
		<div v-if="loading" class="flex min-h-52 items-center justify-center"><n-spin /></div>
		<n-empty v-else-if="issues.length === 0" :description="t('websiteDiagnostic.issueEmpty')" class="py-12" />
		<div v-else class="space-y-3">
			<n-card v-for="issue in issues" :key="issue.id" size="small" hoverable @click="openDetail(issue)">
				<div class="flex cursor-pointer items-start justify-between gap-4">
					<div class="min-w-0">
						<div class="flex flex-wrap items-center gap-2">
							<span class="font-medium text-slate-800">{{ issue.title }}</span>
							<n-tag size="small" :type="statusType(issue.status)">{{ statusText(issue.status) }}</n-tag>
							<n-tag size="small" :type="severityType(issue.severity)">{{ issue.severity }}</n-tag>
						</div>
						<div class="mt-2 break-all font-mono text-xs text-slate-500">{{ issue.route || "-" }}</div>
						<div class="mt-2 text-xs text-slate-500">
							{{ t("websiteDiagnostic.issueCounts", { occurrences: issue.occurrenceCount, sessions: issue.sessionCount }) }}
							· {{ formatTime(issue.lastSeenAt) }}
						</div>
					</div>
					<span class="shrink-0 font-mono text-xs text-slate-400">#{{ issue.id }}</span>
				</div>
			</n-card>
			<n-pagination v-if="total > pageSize" v-model:page="page" :page-size="pageSize" :item-count="total" @update:page="load" />
		</div>

		<n-modal v-model:show="detailVisible" preset="card" style="width: min(760px, 95vw)" :title="t('websiteDiagnostic.issueDetail')">
			<div v-if="detailLoading" class="flex min-h-48 items-center justify-center"><n-spin /></div>
			<n-alert v-else-if="detailError" type="error" :show-icon="true">{{ detailError }}</n-alert>
			<div v-else-if="detail" class="space-y-5">
				<n-descriptions bordered :column="2" size="small">
					<n-descriptions-item :label="t('websiteDiagnostic.issueStatus')">{{ statusText(detail.issue.status) }}</n-descriptions-item>
					<n-descriptions-item :label="t('websiteDiagnostic.issueRelease')">{{ detail.issue.latestRelease || "-" }}</n-descriptions-item>
					<n-descriptions-item :label="t('websiteDiagnostic.issueRoute')"><span class="break-all font-mono text-xs">{{ detail.issue.route || "-" }}</span></n-descriptions-item>
					<n-descriptions-item :label="t('websiteDiagnostic.issueCode')">{{ detail.issue.businessCode || detail.issue.httpStatus || "-" }}</n-descriptions-item>
				</n-descriptions>
				<div>
					<div class="mb-2 text-sm font-semibold">{{ t("websiteDiagnostic.evidence") }}</div>
					<n-empty v-if="detail.events.length === 0" size="small" :description="t('websiteDiagnostic.evidenceEmpty')" />
					<div v-else class="max-h-72 space-y-2 overflow-auto">
						<div v-for="event in detail.events" :key="event.id" class="rounded-lg bg-slate-50 p-3 text-xs">
							<div class="flex gap-2"><n-tag size="small">{{ event.source }}</n-tag><span>{{ formatTime(event.occurredAt) }}</span></div>
							<div class="mt-2 break-all font-mono">{{ event.method }} {{ event.route }} · {{ event.httpStatus || "-" }}</div>
							<pre v-if="event.message || event.stack" class="mt-2 whitespace-pre-wrap break-all text-slate-600">{{ event.message || event.stack }}</pre>
						</div>
					</div>
				</div>
				<div>
					<div class="mb-2 text-sm font-semibold">{{ t("websiteDiagnostic.timeline") }}</div>
					<n-empty v-if="detail.timeline.length === 0" size="small" :description="t('websiteDiagnostic.timelineEmpty')" />
					<n-timeline v-else>
						<n-timeline-item v-for="item in detail.timeline" :key="item.id" :title="item.type" :content="item.content" :time="formatTime(item.createdAt)" />
					</n-timeline>
				</div>
				<div class="flex flex-wrap justify-end gap-2">
					<n-button @click="updateAction('confirm')">{{ t("websiteDiagnostic.confirmIssue") }}</n-button>
					<n-button @click="updateAction('ignore')">{{ t("websiteDiagnostic.ignoreIssue") }}</n-button>
					<n-button @click="updateAction('reopen')">{{ t("websiteDiagnostic.reopenIssue") }}</n-button>
					<n-button v-if="detail.issue.codeSessionId" type="info" @click="openCode">{{ t("websiteDiagnostic.openCode") }}</n-button>
					<n-button type="primary" @click="codeVisible = true">{{ t("websiteDiagnostic.handoffCode") }}</n-button>
					<n-button type="success" @click="verify">{{ t("websiteDiagnostic.startVerify") }}</n-button>
				</div>
			</div>
		</n-modal>

		<n-modal v-model:show="codeVisible" preset="card" style="width: min(560px, 95vw)" :title="t('websiteDiagnostic.handoffCode')">
			<n-form-item :label="t('websiteDiagnostic.codeRequirement')"><n-input v-model:value="requirement" type="textarea" :rows="5" /></n-form-item>
			<n-checkbox v-model:checked="runChecks">{{ t("websiteDiagnostic.runChecks") }}</n-checkbox>
			<template #footer><div class="flex justify-end gap-2"><n-button @click="codeVisible = false">{{ t("websiteDiagnostic.cancel") }}</n-button><n-button type="primary" :loading="actionLoading" @click="handoff">{{ t("websiteDiagnostic.confirmHandoff") }}</n-button></div></template>
		</n-modal>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { useI18n } from "vue-i18n"
import { useMessage } from "naive-ui"
import type { WebsiteIssue, WebsiteIssueDetail, WebsiteIssueStatus } from "@/api/interface/websiteDiagnostic"
import { getWebsiteDiagnosticIssueAPI, handoffWebsiteDiagnosticIssueAPI, listWebsiteDiagnosticIssuesAPI, updateWebsiteDiagnosticIssueAPI, verifyWebsiteDiagnosticIssueAPI } from "@/api/modules/website"
import { formatTime } from "@/utils/date"
import { websiteDiagnosticMessages } from "../websiteDiagnosticMessages"

const props = defineProps<{ websiteId: number; codeProjectId: number }>()
const emit = defineEmits<{ (event: "changed"): void }>()
const { t } = useI18n({ messages: websiteDiagnosticMessages })
const message = useMessage()
const router = useRouter()
const loading = ref(false), detailLoading = ref(false), actionLoading = ref(false)
const error = ref(""), detailError = ref("")
const issues = ref<WebsiteIssue[]>([]), total = ref(0), page = ref(1), pageSize = 10
const status = ref("all"), detailVisible = ref(false), codeVisible = ref(false)
const detail = ref<WebsiteIssueDetail | null>(null), requirement = ref(""), runChecks = ref(true)
const statusOptions = computed(() => ["all", "open", "confirmed", "code_processing", "fix_ready", "verifying", "resolved", "ignored", "reopened"].map(value => ({ label: value === "all" ? t("websiteDiagnostic.statusAll") : statusText(value as WebsiteIssueStatus), value })))
const errorText = (value: unknown, fallback: string) => value instanceof Error && value.message ? value.message : fallback
function statusText(value: WebsiteIssueStatus) { return t(`websiteDiagnostic.status.${value}`) }
function statusType(value: WebsiteIssueStatus): "default" | "warning" | "error" | "info" | "success" { return ({ open: "error", confirmed: "warning", ignored: "default", code_processing: "info", fix_ready: "success", verifying: "info", resolved: "success", reopened: "error" } as const)[value] }
function severityType(value: string): "default" | "warning" | "error" { return value === "critical" || value === "error" ? "error" : value === "warning" ? "warning" : "default" }
async function load() { loading.value = true; error.value = ""; try { const response = await listWebsiteDiagnosticIssuesAPI(props.websiteId, { page: page.value, limit: pageSize, status: status.value }); issues.value = response.data.items || []; total.value = response.data.total || 0 } catch (value) { error.value = errorText(value, t("websiteDiagnostic.issueLoadFailed")); message.error(error.value) } finally { loading.value = false } }
async function openDetail(issue: WebsiteIssue) { detailVisible.value = true; detailLoading.value = true; detailError.value = ""; try { detail.value = (await getWebsiteDiagnosticIssueAPI(props.websiteId, issue.id)).data } catch (value) { detailError.value = errorText(value, t("websiteDiagnostic.issueDetailFailed")); message.error(detailError.value) } finally { detailLoading.value = false } }
async function updateAction(action: "confirm" | "ignore" | "reopen") { if (!detail.value) return; actionLoading.value = true; try { await updateWebsiteDiagnosticIssueAPI(props.websiteId, detail.value.issue.id, action); await openDetail(detail.value.issue); await load(); emit("changed") } catch (value) { message.error(errorText(value, t("websiteDiagnostic.actionFailed"))) } finally { actionLoading.value = false } }
async function handoff() { if (!detail.value || !props.codeProjectId) { message.warning(t("websiteDiagnostic.selectProject")); return } actionLoading.value = true; try { detail.value.issue = (await handoffWebsiteDiagnosticIssueAPI(props.websiteId, detail.value.issue.id, { requirement: requirement.value, allowCode: true, runChecks: runChecks.value })).data; codeVisible.value = false; message.success(t("websiteDiagnostic.handoffCreated")); await load(); emit("changed") } catch (value) { message.error(errorText(value, t("websiteDiagnostic.handoffFailed"))) } finally { actionLoading.value = false } }
async function verify() { if (!detail.value) return; actionLoading.value = true; try { detail.value.issue = (await verifyWebsiteDiagnosticIssueAPI(props.websiteId, detail.value.issue.id, "")).data; message.success(t("websiteDiagnostic.verifyStarted")); await load() } catch (value) { message.error(errorText(value, t("websiteDiagnostic.verifyFailed"))) } finally { actionLoading.value = false } }
function openCode() { if (!detail.value || !props.codeProjectId) return; void router.push({ path: `/code/project/${props.codeProjectId}`, query: { taskId: String(detail.value.issue.codeTaskId) } }) }
onMounted(load)
defineExpose({ load })
</script>
