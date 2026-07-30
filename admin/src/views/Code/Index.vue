<template>
  <div
    class="bg-base-accent border-base-accent rounded-[28px] w-full relative"
    style="min-height: calc(100vh - 130px); height: 100%; display: flex; flex-direction: column;"
  >
    <div class="group-lobby flex-1 overflow-y-auto p-6 md:p-10">
      <div class="lobby-header mb-10 flex justify-between items-center">
        <div>
          <h1 class="text-2xl font-bold text-[var(--n-text-color)] mb-2">{{ $t('code.workspace') }}</h1>
          <p class="text-[var(--n-text-color-3)] text-sm">{{ $t('code.workspaceDesc') }}</p>
        </div>
        <n-button
          type="primary"
          size="large"
          @click="openCreateProjectModal"
          round
        >
          <template #icon>
            <AddIcon />
          </template>
          {{ t("code.createProject") }}
        </n-button>
      </div>

      <div
        v-if="groupsLoading"
        class="flex justify-center items-center h-64"
      >
        <n-spin size="large" />
      </div>

      <n-alert
        v-else-if="groupsLoadError"
        type="error"
        :show-icon="false"
      >
        <div class="flex items-center justify-between gap-3">
          <span>{{ t("code.projectLoadFailed") }}</span>
          <n-button text type="primary" @click="fetchGroups()">{{ t("code.retry") }}</n-button>
        </div>
      </n-alert>

      <div
        v-else-if="groups.length === 0"
        class="flex justify-center items-center h-64"
      >
        <n-empty
          :description="t('code.noProject')"
          size="huge"
        />
      </div>

      <div
        v-else
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
      >
        <div
          v-for="group in groups"
          :key="group.id"
          class="group-card cursor-pointer relative overflow-hidden rounded-3xl p-6"
          @click="enterGroup(group.id)"
        >
          <div class="group-card__glow"></div>
          <div class="group-card__grid"></div>
          <div class="flex justify-between items-start mb-4">
            <div class="group-card__avatar w-12 h-12 rounded-2xl flex items-center justify-center text-xl font-bold">
              {{ group.name.substring(0, 1).toUpperCase() }}
            </div>
            <div class="relative z-[2] flex items-center gap-2" @click.stop>
              <n-button quaternary circle size="small" @click="openEditProjectModal(group)">
                <template #icon><Icon name="mdi:pencil-outline" :size="16" /></template>
              </n-button>
              <n-tag size="small" type="info" round>{{ t("code.project") }}</n-tag>
            </div>
          </div>
          <h3 class="group-card__title text-lg font-semibold mb-2">{{ group.name }}</h3>
          <p class="group-card__desc text-sm line-clamp-2 mb-5">{{ group.description || $t('code.noDesc') }}</p>
          <div
            class="group-card__path mb-4 flex items-center gap-2 text-xs"
            :title="group.sourceDirs?.join('\n') || group.workDir"
          >
            <Icon name="mdi:folder-outline" :size="16" />
            <span class="truncate">
              {{ group.sourceDirs?.length ? t("code.projectDirectoryCount", { count: group.sourceDirs.length }) : group.workDir || t("code.projectDirectoryRequired") }}
            </span>
          </div>
          <div class="group-card__status mb-4 rounded-2xl border border-[var(--n-border-color)] bg-[var(--n-color-embedded)] p-3">
            <div class="flex items-center justify-between gap-3">
              <div class="flex min-w-0 items-center gap-2">
                <span
                  class="h-2 w-2 shrink-0 rounded-full"
                  :class="projectStatusMeta(group).dotClass"
                ></span>
                <span class="truncate text-sm font-semibold text-[var(--n-text-color)]">
                  {{ t(projectStatusMeta(group).labelKey) }}
                </span>
              </div>
              <span v-if="group.executionSummary.updatedAt" class="shrink-0 text-[11px] text-[var(--n-text-color-3)]">
                {{ formatUpdatedAt(group.executionSummary.updatedAt) }}
              </span>
            </div>
            <div class="mt-2 truncate text-xs text-[var(--n-text-color-2)]" :title="group.executionSummary.currentTaskTitle">
              {{ group.executionSummary.currentTaskTitle || t("code.projectIdleHint") }}
            </div>
            <div v-if="group.executionSummary.activeTaskCount" class="mt-2 flex flex-wrap gap-x-3 gap-y-1 text-[11px] text-[var(--n-text-color-3)]">
              <span>{{ t("code.activeTaskCount", { count: group.executionSummary.activeTaskCount }) }}</span>
              <span v-if="group.executionSummary.pendingApprovalCount" class="text-orange-500">
                {{ t("code.pendingApprovalCount", { count: group.executionSummary.pendingApprovalCount }) }}
              </span>
            </div>
          </div>
          <div class="group-card__footer flex justify-between items-center text-xs pt-4">
            <span>{{ group.taskCount || 0 }} {{ $t('code.task') }}</span>
            <div class="relative z-[2] flex items-center gap-3" @click.stop>
              <n-button text type="primary" size="small" @click="openQuickPanel(group)">
                <template #icon><Icon name="mdi:dock-window" :size="16" /></template>
                {{ t("code.quickPanel") }}
              </n-button>
              <span class="group-card__action cursor-pointer" @click="enterGroup(group.id)">{{ t("code.enterProject") }}</span>
            </div>
          </div>
        </div>
      </div>
      <n-modal
        v-model:show="showCreateProjectModal"
        preset="dialog"
        :title="editingProjectId ? t('code.editProject') : t('code.createProject')"
      >
        <div class="flex flex-col gap-4 mt-4">
          <n-input
            v-model:value="projectForm.name"
            :placeholder="t('code.projectName')"
            placeholder-class="text-[var(--n-text-color-3)]"
          />
          <n-input
            v-model:value="projectForm.desc"
            type="textarea"
            :placeholder="t('code.projectDesc')"
          />
          <div>
            <div class="mb-2 flex items-center justify-between gap-3">
              <div class="text-sm font-medium text-[var(--n-text-color)]">{{ t("code.projectDirectories") }}</div>
              <n-button type="primary" secondary size="small" @click="showDirectoryPicker = true">
                {{ t("code.browseDirectory") }}
              </n-button>
            </div>
            <div v-if="projectForm.sourceDirs.length" class="flex flex-wrap gap-2 rounded-xl bg-[var(--n-color-embedded)] p-3">
              <n-tag
                v-for="sourceDir in projectForm.sourceDirs"
                :key="sourceDir"
                closable
                :title="sourceDir"
                @close="removeSourceDir(sourceDir)"
              >{{ sourceDir }}</n-tag>
            </div>
            <n-empty v-else size="small" :description="t('code.projectDirectoryRequired')" />
            <div class="mt-2 text-xs text-[var(--n-text-color-3)]">{{ t("code.projectDirectoriesHint") }}</div>
          </div>
        </div>
        <template #action>
          <n-button :disabled="creatingProject" @click="showCreateProjectModal = false">
            {{ $t('commons.button.cancel') }}
          </n-button>
          <n-button
            type="primary"
            :loading="creatingProject"
            @click="submitProject"
          >{{ editingProjectId ? t("code.saveChanges") : $t('commons.button.confirm') }}</n-button>
        </template>
      </n-modal>
      <ProjectDirectoryPicker
        v-model:show="showDirectoryPicker"
		:initial-path="projectForm.workDir || defaultWorkDir"
		:root-path="directoryRoot"
		:selected-paths="projectForm.sourceDirs"
		@select="projectForm.sourceDirs = $event"
      />
      <ProjectQuickPanels ref="quickPanelsRef" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAIGroups, createAIGroup, updateAIGroup } from '@/api/modules/code'
import type { AIGroup } from '@/api/interface/code'
import Icon from '@/components/common/Icon.vue'
import ProjectDirectoryPicker from './components/ProjectDirectoryPicker.vue'
import ProjectQuickPanels from './components/ProjectQuickPanels.vue'
import { codeProjectMessages } from '@/i18n/locales/codeProject'

const AddIcon = () => '+'

const message = useMessage()
const router = useRouter()
const { t } = useI18n({ messages: codeProjectMessages })

const showCreateProjectModal = ref(false)
const showDirectoryPicker = ref(false)
const creatingProject = ref(false)
const editingProjectId = ref<number | null>(null)
const projectForm = ref({ name: '', desc: '', workDir: '', sourceDirs: [] as string[] })

const groups = ref<AIGroup[]>([])
const groupsLoading = ref(false)
const groupsLoadError = ref(false)
const groupsRefreshing = ref(false)
const defaultWorkDir = ref("/")
const directoryRoot = ref("/")
const quickPanelsRef = ref<InstanceType<typeof ProjectQuickPanels> | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | undefined

const fetchGroups = async (silent = false) => {
  if (groupsRefreshing.value) return
  groupsRefreshing.value = true
  if (!silent) {
    groupsLoading.value = true
    groupsLoadError.value = false
  }
  try {
    const res = await getAIGroups({ page: 1, limit: 50 })
    if (res.code === 0) {
      groups.value = res.data.items || []
      const directoryDefaults = res.data as typeof res.data & { defaultWorkDir?: string; directoryRoot?: string }
      defaultWorkDir.value = directoryDefaults.defaultWorkDir || "/"
      directoryRoot.value = directoryDefaults.directoryRoot || "/"
    }
  } catch {
    if (!silent) {
      groupsLoadError.value = true
      groups.value = []
    }
  } finally {
    if (!silent) groupsLoading.value = false
    groupsRefreshing.value = false
  }
}

onMounted(() => {
  fetchGroups()
  refreshTimer = setInterval(() => fetchGroups(true), 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

const projectStatusMeta = (group: AIGroup) => {
  const status = group.executionSummary.pendingApprovalCount > 0 ? "pending_approval" : group.executionSummary.status
  return {
    idle: { labelKey: "code.projectStatus_idle", dotClass: "bg-slate-400" },
    queued: { labelKey: "code.projectStatus_queued", dotClass: "bg-blue-400 animate-pulse" },
    running: { labelKey: "code.projectStatus_running", dotClass: "bg-emerald-500 animate-pulse" },
    pending_approval: { labelKey: "code.projectStatus_pendingApproval", dotClass: "bg-orange-500 animate-pulse" }
  }[status] || { labelKey: "code.projectStatus_idle", dotClass: "bg-slate-400" }
}

const formatUpdatedAt = (value: string) => new Date(value).toLocaleString(undefined, {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit"
})

const openCreateProjectModal = () => {
  editingProjectId.value = null
  projectForm.value = { name: '', desc: '', workDir: defaultWorkDir.value, sourceDirs: [] }
  showCreateProjectModal.value = true
}

const openEditProjectModal = (project: AIGroup) => {
  editingProjectId.value = project.id
  const sourceDirs = project.sourceDirs?.length ? project.sourceDirs : project.workDir ? [project.workDir] : []
  projectForm.value = { name: project.name, desc: project.description || '', workDir: sourceDirs[0] || defaultWorkDir.value, sourceDirs }
  showCreateProjectModal.value = true
}

const removeSourceDir = (sourceDir: string) => {
  projectForm.value.sourceDirs = projectForm.value.sourceDirs.filter(path => path !== sourceDir)
}

const submitProject = async () => {
  if (!projectForm.value.name.trim()) {
    message.warning(t('code.projectNameRequired'))
    return
  }
  if (projectForm.value.sourceDirs.length === 0) {
    message.warning(t('code.projectDirectoryRequired'))
    return
  }
  creatingProject.value = true
  try {
    const payload = {
      name: projectForm.value.name.trim(),
      description: projectForm.value.desc.trim(),
      sourceDirs: projectForm.value.sourceDirs
    }
    const res = editingProjectId.value
      ? await updateAIGroup(editingProjectId.value, payload)
      : await createAIGroup(payload)
    if (res.code === 0) {
      showCreateProjectModal.value = false
      message.success(t(editingProjectId.value ? 'code.projectUpdateSuccess' : 'code.projectCreateSuccess'))
      await fetchGroups()
    }
  } catch {
    message.error(t(editingProjectId.value ? 'code.projectUpdateFailed' : 'code.projectCreateFailed'))
  } finally {
    creatingProject.value = false
  }
}

const enterGroup = (id: number) => {
  router.push(`/code/group/${id}`)
}

const openQuickPanel = (project: AIGroup) => quickPanelsRef.value?.open(project)
</script>

<style scoped>
.group-card {
  background:
    radial-gradient(circle at top right, color-mix(in srgb, var(--n-primary-color) 16%, transparent), transparent 34%),
    linear-gradient(180deg, color-mix(in srgb, var(--n-color) 98%, white 2%), color-mix(in srgb, var(--n-color) 94%, black 6%));
  border: 1px solid color-mix(in srgb, var(--n-border-color) 88%, var(--n-primary-color) 12%);
  box-shadow:
    0 12px 28px rgba(15, 23, 42, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
  transition:
    transform 0.28s ease,
    box-shadow 0.28s ease,
    border-color 0.28s ease,
    background 0.28s ease;
}

.group-card:hover {
  transform: translateY(-6px);
  border-color: color-mix(in srgb, var(--n-primary-color) 42%, var(--n-border-color) 58%);
  box-shadow:
    0 20px 40px rgba(15, 23, 42, 0.12),
    0 8px 18px rgba(59, 130, 246, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.72);
}

.group-card__glow {
  position: absolute;
  top: -44px;
  right: -34px;
  width: 130px;
  height: 130px;
  border-radius: 9999px;
  background: color-mix(in srgb, var(--n-primary-color) 20%, transparent);
  filter: blur(18px);
  opacity: 0.9;
  pointer-events: none;
}

.group-card__grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
  background-size: 18px 18px;
  mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.28), transparent 72%);
  pointer-events: none;
}

.group-card__avatar,
.group-card__title,
.group-card__desc,
.group-card__path,
.group-card__footer {
  position: relative;
  z-index: 1;
}

.group-card__avatar {
  color: var(--n-primary-color);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--n-primary-color) 18%, white 82%), color-mix(in srgb, var(--n-primary-color) 6%, transparent));
  border: 1px solid color-mix(in srgb, var(--n-primary-color) 22%, transparent);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.7),
    0 8px 16px rgba(59, 130, 246, 0.12);
}

.group-card__title {
  color: var(--n-text-color);
  letter-spacing: -0.01em;
}

.group-card__desc {
  color: var(--n-text-color-3);
  line-height: 1.65;
  min-height: 48px;
}

.group-card__path {
  color: var(--n-text-color-3);
}

.group-card__status {
  position: relative;
  z-index: 1;
}

.group-card__footer {
  color: var(--n-text-color-3);
  border-top: 1px solid color-mix(in srgb, var(--n-border-color) 82%, transparent);
}

.group-card__action {
  color: var(--n-primary-color);
  font-weight: 600;
  transition: transform 0.28s ease;
}

.group-card:hover .group-card__action {
  transform: translateX(4px);
}
</style>
