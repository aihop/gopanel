<template>
  <div
    class="bg-base-accent border-base-accent rounded-[28px] w-full relative"
    style="min-height: calc(100vh - 130px); height: 100%; display: flex; flex-direction: column;"
  >
    <div class="project-lobby flex-1 overflow-y-auto p-6 md:p-10">
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
        v-if="projectsLoading"
        class="flex justify-center items-center h-64"
      >
        <n-spin size="large" />
      </div>

      <n-alert
        v-else-if="projectsLoadError"
        type="error"
        :show-icon="false"
      >
        <div class="flex items-center justify-between gap-3">
          <span>{{ t("code.projectLoadFailed") }}</span>
          <n-button text type="primary" @click="fetchProjects()">{{ t("code.retry") }}</n-button>
        </div>
      </n-alert>

      <div
        v-else-if="projects.length === 0"
        class="flex justify-center items-center h-64"
      >
        <n-empty
          :description="t('code.noProject')"
          size="huge"
        />
      </div>

      <div
        v-else
        class="grid grid-cols-1 gap-5 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
      >
        <div
          v-for="project in projects"
          :key="project.id"
          class="project-card relative flex cursor-pointer flex-col overflow-hidden rounded-[24px] p-5"
          :class="`project-card--${projectStatusMeta(project).status}`"
          @click="enterProject(project.id)"
        >
          <div class="mb-5 flex items-start justify-between gap-3">
            <div class="flex min-w-0 items-center gap-3.5">
              <div class="project-card__avatar flex h-11 w-11 shrink-0 items-center justify-center rounded-[14px] text-lg font-bold">
                {{ project.name.substring(0, 1).toUpperCase() }}
              </div>
              <div class="min-w-0">
                <h3 class="project-card__title truncate text-lg font-semibold">{{ project.name }}</h3>
                <div class="mt-1 flex items-center gap-1.5 text-xs text-[var(--n-text-color-3)]">
                  <Icon name="mdi:folder-outline" :size="15" />
                  <span class="truncate">
                    {{ project.sourceDirs?.length ? t("code.projectDirectoryCount", { count: project.sourceDirs.length }) : project.workDir || t("code.projectDirectoryRequired") }}
                  </span>
                </div>
              </div>
            </div>
            <div class="relative z-[2] shrink-0" @click.stop>
              <n-button quaternary circle size="small" :aria-label="t('code.editProject')" @click="openEditProjectModal(project)">
                <template #icon><Icon name="mdi:pencil-outline" :size="16" /></template>
              </n-button>
            </div>
          </div>
          <p class="project-card__desc mb-5 line-clamp-2 min-h-11 text-sm">{{ project.description || $t('code.noDesc') }}</p>
          <div class="project-card__status mb-5 rounded-[18px] p-3.5">
            <div class="flex items-center justify-between gap-3">
              <div class="flex min-w-0 items-center gap-2.5">
                <span class="project-card__status-icon flex h-8 w-8 shrink-0 items-center justify-center rounded-xl">
                  <Icon :name="projectStatusMeta(project).icon" :size="17" />
                </span>
                <span class="truncate text-sm font-semibold">
                  {{ t(projectStatusMeta(project).labelKey) }}
                </span>
              </div>
              <span v-if="project.executionSummary.updatedAt" class="shrink-0 text-[11px] text-[var(--n-text-color-3)]">
                {{ formatUpdatedAt(project.executionSummary.updatedAt) }}
              </span>
            </div>
            <div class="mt-3 truncate text-xs text-[var(--n-text-color-2)]" :title="project.executionSummary.currentTaskTitle">
              {{ project.executionSummary.currentTaskTitle || t("code.projectIdleHint") }}
            </div>
            <div v-if="project.executionSummary.activeTaskCount" class="mt-3 flex flex-wrap gap-2 text-[11px]">
              <span class="rounded-full bg-black/5 px-2 py-1 text-[var(--n-text-color-3)] dark:bg-white/10">
                {{ t("code.activeTaskCount", { count: project.executionSummary.activeTaskCount }) }}
              </span>
              <span v-if="project.executionSummary.pendingApprovalCount" class="rounded-full bg-amber-500/10 px-2 py-1 font-medium text-amber-600 dark:text-amber-400">
                {{ t("code.pendingApprovalCount", { count: project.executionSummary.pendingApprovalCount }) }}
              </span>
            </div>
          </div>
          <div class="project-card__footer mt-auto flex items-center justify-between gap-3 pt-4 text-xs">
            <span class="flex items-center gap-1.5">
              <Icon name="mdi:checkbox-marked-circle-outline" :size="15" />
              {{ project.taskCount || 0 }} {{ $t('code.task') }}
            </span>
            <div class="relative z-[2] flex items-center gap-2" @click.stop>
              <n-button quaternary circle type="primary" size="small" :title="t('code.quickPanel')" :aria-label="t('code.quickPanel')" @click="openQuickPanel(project)">
                <template #icon><Icon name="mdi:dock-window" :size="16" /></template>
              </n-button>
              <n-button text type="primary" size="small" @click="enterProject(project.id)">
                <template #icon><Icon name="mdi:arrow-right" :size="16" /></template>
                {{ t("code.enterProject") }}
              </n-button>
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
		  <div class="rounded-xl bg-[var(--n-color-embedded)] p-3">
			<div class="flex items-start justify-between gap-3">
				<div><div class="text-sm font-medium">{{ t("code.requireQualityGate") }}</div><div class="mt-1 text-xs text-[var(--n-text-color-3)]">{{ t("code.requireQualityGateHint") }}</div></div>
				<n-switch :value="true" disabled />
			</div>
		  </div>
		  <n-input-number v-model:value="projectForm.monthlyTokenBudget" :min="0" :step="100000" style="width: 100%" :placeholder="t('code.monthlyTokenBudget')">
			<template #prefix>{{ t("code.monthlyTokenBudget") }}</template>
		  </n-input-number>
		  <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">{{ t("code.monthlyTokenBudgetHint") }}</div>
		  <n-select v-model:value="projectForm.primaryRepository" :options="primaryRepositoryOptions" :loading="repositoriesLoading" clearable :placeholder="t('code.primaryRepositoryPlaceholder')" />
		  <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">{{ t("code.primaryRepositoryHint") }}</div>
		  <n-input v-model:value="projectForm.deliveryBranch" :placeholder="t('code.deliveryBranchPlaceholder')">
			<template #prefix>{{ t("code.deliveryBranch") }}</template>
		  </n-input>
		  <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">{{ t("code.deliveryBranchHint") }}</div>
		  <ProjectGitCredentialSelect v-model="projectForm.gitCredentialId" />
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
		@select="handleSourceDirsSelected"
      />
      <ProjectQuickPanels ref="quickPanelsRef" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAIProjects, createAIProject, discoverCodeProjectRepositories, updateAIProject } from '@/api/modules/code'
import type { AIProject } from '@/api/interface/code'
import Icon from '@/components/common/Icon.vue'
import ProjectDirectoryPicker from './components/ProjectDirectoryPicker.vue'
import ProjectGitCredentialSelect from './components/ProjectGitCredentialSelect.vue'
import ProjectQuickPanels from './components/ProjectQuickPanels.vue'
import { codeProjectMessages } from '@/i18n/locales/codeProject'

defineOptions({ name: "CodeIndex" })

const AddIcon = () => '+'

const message = useMessage()
const router = useRouter()
const { t } = useI18n({ messages: codeProjectMessages })

const showCreateProjectModal = ref(false)
const showDirectoryPicker = ref(false)
const creatingProject = ref(false)
const repositoriesLoading = ref(false)
const repositoryOptions = ref<Array<{ label: string; value: string }>>([])
const editingProjectId = ref<number | null>(null)
const projectForm = ref({ name: '', desc: '', workDir: '', sourceDirs: [] as string[], primaryRepository: '', deliveryBranch: '', gitCredentialId: 0, requireQualityGate: true, monthlyTokenBudget: 0 })

const projects = ref<AIProject[]>([])
const projectsLoading = ref(false)
const projectsLoadError = ref(false)
const projectsRefreshing = ref(false)
const defaultWorkDir = ref("/")
const directoryRoot = ref("/")
const quickPanelsRef = ref<InstanceType<typeof ProjectQuickPanels> | null>(null)
let refreshTimer: ReturnType<typeof setInterval> | undefined

const primaryRepositoryOptions = computed(() => [
  { label: t('code.primaryRepositoryAuto'), value: '' },
  ...repositoryOptions.value,
])

const loadRepositoryOptions = async () => {
  const sourceDirs = projectForm.value.sourceDirs
  if (!sourceDirs.length) { repositoryOptions.value = []; projectForm.value.primaryRepository = ''; return }
  repositoriesLoading.value = true
  try {
    const res = await discoverCodeProjectRepositories(sourceDirs)
    repositoryOptions.value = (res.data || []).map(item => ({ label: `${item.name} · ${item.path}`, value: item.path }))
    if (projectForm.value.primaryRepository && !repositoryOptions.value.some(item => item.value === projectForm.value.primaryRepository)) {
      projectForm.value.primaryRepository = ''
    }
  } catch {
    repositoryOptions.value = []
    projectForm.value.primaryRepository = ''
  } finally {
    repositoriesLoading.value = false
  }
}

const fetchProjects = async (silent = false) => {
  if (projectsRefreshing.value) return
  projectsRefreshing.value = true
  if (!silent) {
    projectsLoading.value = true
    projectsLoadError.value = false
  }
  try {
    const res = await getAIProjects({ page: 1, limit: 50 })
    if (res.code === 0) {
      projects.value = res.data.items || []
      const directoryDefaults = res.data as typeof res.data & { defaultWorkDir?: string; directoryRoot?: string }
      defaultWorkDir.value = directoryDefaults.defaultWorkDir || "/"
      directoryRoot.value = directoryDefaults.directoryRoot || "/"
    }
  } catch {
    if (!silent) {
      projectsLoadError.value = true
      projects.value = []
    }
  } finally {
    if (!silent) projectsLoading.value = false
    projectsRefreshing.value = false
  }
}

onMounted(() => {
  fetchProjects()
  refreshTimer = setInterval(() => fetchProjects(true), 10000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

const projectStatusMeta = (project: AIProject) => {
  const status = project.executionSummary.pendingApprovalCount > 0 ? "pending_approval" : project.executionSummary.status
  return {
    idle: { status: "idle", labelKey: "code.projectStatus_idle", icon: "mdi:circle-slice-8" },
	queued: { status: "queued", labelKey: "code.projectStatus_queued", icon: "mdi:progress-clock" },
	running: { status: "running", labelKey: "code.projectStatus_running", icon: "mdi:play-circle-outline" },
	delivering: { status: "running", labelKey: "code.projectStatus_delivering", icon: "mdi:source-merge" },
	pending_approval: { status: "pending-approval", labelKey: "code.projectStatus_pendingApproval", icon: "mdi:shield-alert-outline" },
  }[status] || { status: "idle", labelKey: "code.projectStatus_idle", icon: "mdi:circle-slice-8" }
}

const formatUpdatedAt = (value: string) => new Date(value).toLocaleString(undefined, {
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
	minute: "2-digit",
})

const openCreateProjectModal = () => {
  editingProjectId.value = null
  projectForm.value = { name: '', desc: '', workDir: defaultWorkDir.value, sourceDirs: [], primaryRepository: '', deliveryBranch: '', gitCredentialId: 0, requireQualityGate: true, monthlyTokenBudget: 0 }
  showCreateProjectModal.value = true
  repositoryOptions.value = []
}

const openEditProjectModal = (project: AIProject) => {
  editingProjectId.value = project.id
  const sourceDirs = project.sourceDirs?.length ? project.sourceDirs : project.workDir ? [project.workDir] : []
  projectForm.value = { name: project.name, desc: project.description || '', workDir: sourceDirs[0] || defaultWorkDir.value, sourceDirs, primaryRepository: project.primaryRepository || '', deliveryBranch: project.deliveryBranch || 'main', gitCredentialId: project.gitCredentialId || 0, requireQualityGate: true, monthlyTokenBudget: project.monthlyTokenBudget || 0 }
  showCreateProjectModal.value = true
  void loadRepositoryOptions()
}

const removeSourceDir = (sourceDir: string) => {
  projectForm.value.sourceDirs = projectForm.value.sourceDirs.filter(path => path !== sourceDir)
  void loadRepositoryOptions()
}

const handleSourceDirsSelected = (sourceDirs: string[]) => {
  projectForm.value.sourceDirs = sourceDirs
  void loadRepositoryOptions()
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
	  sourceDirs: projectForm.value.sourceDirs,
	  primaryRepository: projectForm.value.primaryRepository?.trim() || '',
	  deliveryBranch: projectForm.value.deliveryBranch.trim(),
	  gitCredentialId: projectForm.value.gitCredentialId,
      requireQualityGate: projectForm.value.requireQualityGate,
      monthlyTokenBudget: projectForm.value.monthlyTokenBudget || 0,
    }
    const res = editingProjectId.value
      ? await updateAIProject(editingProjectId.value, payload)
      : await createAIProject(payload)
    if (res.code === 0) {
      showCreateProjectModal.value = false
      message.success(t(editingProjectId.value ? 'code.projectUpdateSuccess' : 'code.projectCreateSuccess'))
      await fetchProjects()
    }
  } catch {
    void 0
  } finally {
    creatingProject.value = false
  }
}

const enterProject = (id: number) => {
  router.push(`/code/project/${id}`)
}

const openQuickPanel = (project: AIProject) => quickPanelsRef.value?.open(project)
</script>

<style scoped>
.project-card {
  --status-accent: #94a3b8;
  background: color-mix(in srgb, var(--n-color) 97%, transparent);
  border: 1px solid color-mix(in srgb, var(--n-border-color) 92%, transparent);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.045);
  transition:
    transform 0.22s ease,
    box-shadow 0.22s ease,
    border-color 0.22s ease;
}

.project-card::before {
  position: absolute;
  inset: 24px auto 24px 0;
  width: 3px;
  border-radius: 0 999px 999px 0;
  background: var(--n-primary-color);
  content: "";
}

.project-card--queued {
  --status-accent: #3b82f6;
}

.project-card--running {
  --status-accent: #10b981;
}

.project-card--pending-approval {
  --status-accent: #f59e0b;
}

.project-card:hover {
  transform: translateY(-3px);
  border-color: color-mix(in srgb, var(--n-primary-color) 34%, var(--n-border-color));
  box-shadow: 0 14px 34px rgba(15, 23, 42, 0.085);
}

.project-card__avatar {
  color: var(--n-primary-color);
  background: color-mix(in srgb, var(--n-primary-color) 10%, var(--n-color));
  border: 1px solid color-mix(in srgb, var(--n-primary-color) 20%, transparent);
}

.project-card__title {
  color: var(--n-text-color);
  letter-spacing: -0.01em;
}

.project-card__desc {
  color: var(--n-text-color-3);
  line-height: 1.6;
}

.project-card__status {
  color: var(--status-accent);
  background: color-mix(in srgb, var(--n-color-embedded) 92%, var(--n-color));
  border: 1px solid color-mix(in srgb, var(--n-border-color) 82%, transparent);
}

.project-card__status-icon {
  color: var(--status-accent);
  background: color-mix(in srgb, var(--status-accent) 10%, var(--n-color));
  border: 1px solid color-mix(in srgb, var(--status-accent) 16%, transparent);
}

.project-card__footer {
  color: var(--n-text-color-3);
  border-top: 1px solid color-mix(in srgb, var(--n-border-color) 76%, transparent);
}
</style>
