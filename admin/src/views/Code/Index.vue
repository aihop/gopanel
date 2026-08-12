<template>
  <!--
    用布局自带的 page-wrapped 拿确定高度（100svh - 工具栏 - 视图内边距），
    不要手写 min-height：祖先链没有确定高度时 height:100% 会失效，
    flex 链就没有可分配的高度，终端只能被最小高度撑着，怎么调间距都不管用。
  -->
  <div
    class="page page-wrapped page-mobile-full page-without-footer bg-base-accent border-base-accent relative flex w-full flex-col overflow-hidden"
  >
    <!-- 整页不滚：右侧是终端，页面滚起来终端会跟着跑。滚动交给左右两栏各自处理。 -->
    <!-- 横向可以给足，纵向省着用：只有纵向内边距会从终端高度里扣 -->
    <div class="project-lobby flex min-h-0 flex-1 flex-col overflow-hidden pt-4 md:pt-5">
      <CodeDashboard
        :projects="projects"
        :loading="projectsLoading"
        :load-error="projectsLoadError"
        @retry="fetchProjects()"
        @create-project="openCreateProjectModal"
        @create-task="openNewProjectTask"
        @project-action="handleProjectAction"
        @open-task="openTask"
      >
        <!--
          项目管理入口。刻意不做成任务列表的筛选器：
          首页列的是「所有在做的任务」，按项目切会把这个口径打散。
          这里只负责进入 / 浮窗 / 编辑，不影响下面的列表。
        -->
        <template #toolbar>
          <n-dropdown
            v-if="projects.length"
            trigger="click"
            :options="projectMenuOptions"
            @select="handleProjectMenuSelect"
          >
            <n-button
              size="small"
              quaternary
            >
              <template #icon>
                <Icon
                  name="mdi:folder-multiple-outline"
                  :size="16"
                />
              </template>
              {{ t("code.project") }}
            </n-button>
          </n-dropdown>
          <n-button
            type="primary"
            size="small"
            @click="openCreateProjectModal"
          >
            <template #icon>
              <AddIcon />
            </template>
            {{ t("code.createProject") }}
          </n-button>
        </template>
      </CodeDashboard>

      <n-modal
        v-model:show="showCreateProjectModal"
        preset="dialog"
        style="width: min(760px, 94vw)"
        :title="editingProjectId ? t('code.editProject') : t('code.createProject')"
      >
        <div class="flex flex-col gap-4 mt-4">
          <n-input
            v-model:value="projectForm.name"
            :placeholder="t('code.projectName')"
            placeholder-class="text-[var(--n-text-color-3)]"
          />
          <ProjectQualityGateSettings
            v-model="projectForm.requireQualityGate"
            :checks="projectForm.qualityChecks"
            :source-dirs="projectForm.sourceDirs"
            :repositories="repositoryOptions"
            @update:checks="projectForm.qualityChecks = $event"
          />
          <n-input-number
            v-model:value="projectForm.monthlyTokenBudget"
            :min="0"
            :step="100000"
            style="width: 100%"
            :placeholder="t('code.monthlyTokenBudget')"
          >
            <template #prefix>
              {{ t("code.monthlyTokenBudget") }}
            </template>
          </n-input-number>
          <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
            {{ t("code.monthlyTokenBudgetHint") }}
          </div>
          <n-select
            v-model:value="projectForm.primaryRepository"
            :options="primaryRepositoryOptions"
            :loading="repositoriesLoading"
            clearable
            :placeholder="t('code.primaryRepositoryPlaceholder')"
          />
          <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
            {{ t("code.primaryRepositoryHint") }}
          </div>
          <!--
            复用已经发现到的仓库列表：项目目录本身是仓库、里面又嵌了子仓库时
            （如 app/themes/*），发现逻辑会沿 gitlink 递归全部带进来。
            这里让用户把不参与开发的摘掉，主交付仓库不允许排除。
          -->
          <div v-if="repositoryOptions.length > 1">
            <div class="mb-2 text-sm font-medium text-[var(--n-text-color)]">
              {{ t("code.includedRepositories") }}
            </div>
            <div class="flex flex-col gap-2 rounded-xl bg-[var(--n-color-embedded)] p-3">
              <n-checkbox
                v-for="option in repositoryOptions"
                :key="option.value"
                :checked="!projectForm.excludedRepositories.includes(option.value)"
                :disabled="option.value === projectForm.primaryRepository"
                @update:checked="checked => toggleRepositoryIncluded(option.value, checked)"
              >
                <span class="text-xs">{{ option.label }}</span>
              </n-checkbox>
            </div>
            <div class="mt-2 text-xs text-[var(--n-text-color-3)]">
              {{ t("code.includedRepositoriesHint") }}
            </div>
          </div>
          <n-input
            v-model:value="projectForm.deliveryBranch"
            :placeholder="t('code.deliveryBranchPlaceholder')"
          >
            <template #prefix>
              {{ t("code.deliveryBranch") }}
            </template>
          </n-input>
          <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
            {{ t("code.deliveryBranchHint") }}
          </div>
          <n-select
            v-model:value="projectForm.deliveryMode"
            :options="deliveryModeOptions"
            :placeholder="t('code.deliveryMode')"
          />
          <div class="-mt-3 text-xs text-[var(--n-text-color-3)]">
            {{ projectForm.deliveryMode === "branch" ? t("code.deliveryModeBranchHint") : t("code.deliveryModeDirectHint") }}
          </div>
          <ProjectGitCredentialSelect v-model="projectForm.gitCredentialId" />
          <n-input
            v-model:value="projectForm.desc"
            type="textarea"
            :placeholder="t('code.projectDesc')"
          />
          <div>
            <div class="mb-2 flex items-center justify-between gap-3">
              <div class="text-sm font-medium text-[var(--n-text-color)]">
                {{ t("code.projectDirectories") }}
              </div>
              <n-button
                type="primary"
                secondary
                size="small"
                @click="showDirectoryPicker = true"
              >
                {{ t("code.browseDirectory") }}
              </n-button>
            </div>
            <div
              v-if="projectForm.sourceDirs.length"
              class="flex flex-wrap gap-2 rounded-xl bg-[var(--n-color-embedded)] p-3"
            >
              <n-tag
                v-for="sourceDir in projectForm.sourceDirs"
                :key="sourceDir"
                closable
                :title="sourceDir"
                @close="removeSourceDir(sourceDir)"
              >
                {{ sourceDir }}
              </n-tag>
            </div>
            <n-empty
              v-else
              size="small"
              :description="t('code.projectDirectoryRequired')"
            />
            <div class="mt-2 text-xs text-[var(--n-text-color-3)]">
              {{ t("code.projectDirectoriesHint") }}
            </div>
          </div>
        </div>
        <template #action>
          <n-button
            :disabled="creatingProject"
            @click="showCreateProjectModal = false"
          >
            {{ $t('commons.button.cancel') }}
          </n-button>
          <n-button
            type="primary"
            :loading="creatingProject"
            @click="submitProject"
          >
            {{ editingProjectId ? t("code.saveChanges") : $t('commons.button.confirm') }}
          </n-button>
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
import { computed, h, ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAIProjects, createAIProject, discoverCodeProjectRepositories, updateAIProject } from '@/api/modules/code'
import type { AIProject, CodeProjectQualityCheck } from '@/api/interface/code'
import type { CodeTaskListItem } from '@/api/interface/codeTasks'
import Icon from '@/components/common/Icon.vue'
import CodeProjectIdentity from './components/CodeProjectIdentity.vue'
import CodeDashboard from './components/CodeDashboard.vue'
import ProjectDirectoryPicker from './components/ProjectDirectoryPicker.vue'
import ProjectGitCredentialSelect from './components/ProjectGitCredentialSelect.vue'
import ProjectQuickPanels from './components/ProjectQuickPanels.vue'
import ProjectQualityGateSettings from './components/ProjectQualityGateSettings.vue'
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
const emptyQualityChecks = (): CodeProjectQualityCheck[] => []
const projectForm = ref({ name: '', desc: '', workDir: '', sourceDirs: [] as string[], excludedRepositories: [] as string[], primaryRepository: '', deliveryBranch: '', deliveryMode: 'direct' as 'direct' | 'branch', gitCredentialId: 0, requireQualityGate: true, qualityChecks: emptyQualityChecks(), monthlyTokenBudget: 0 })

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

const deliveryModeOptions = computed(() => [
  { label: t('code.deliveryModeDirect'), value: 'direct' },
  { label: t('code.deliveryModeBranch'), value: 'branch' },
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

const openCreateProjectModal = () => {
  editingProjectId.value = null
  projectForm.value = { name: '', desc: '', workDir: defaultWorkDir.value, sourceDirs: [], excludedRepositories: [], primaryRepository: '', deliveryBranch: '', deliveryMode: 'direct' as 'direct' | 'branch', gitCredentialId: 0, requireQualityGate: true, qualityChecks: emptyQualityChecks(), monthlyTokenBudget: 0 }
  showCreateProjectModal.value = true
  repositoryOptions.value = []
}

const openEditProjectModal = (project: AIProject) => {
  editingProjectId.value = project.id
  const sourceDirs = project.sourceDirs?.length ? project.sourceDirs : project.workDir ? [project.workDir] : []
  projectForm.value = { name: project.name, desc: project.description || '', workDir: sourceDirs[0] || defaultWorkDir.value, sourceDirs, excludedRepositories: [...(project.excludedRepositories || [])], primaryRepository: project.primaryRepository || '', deliveryBranch: project.deliveryBranch || 'main', deliveryMode: project.deliveryMode === 'branch' ? 'branch' : 'direct', gitCredentialId: project.gitCredentialId || 0, requireQualityGate: project.requireQualityGate, qualityChecks: (project.qualityChecks || []).map(check => ({ ...check })), monthlyTokenBudget: project.monthlyTokenBudget || 0 }
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
	  excludedRepositories: projectForm.value.excludedRepositories,
	  primaryRepository: projectForm.value.primaryRepository?.trim() || '',
	  deliveryBranch: projectForm.value.deliveryBranch.trim(),
	  deliveryMode: projectForm.value.deliveryMode,
	  gitCredentialId: projectForm.value.gitCredentialId,
      requireQualityGate: projectForm.value.requireQualityGate,
	  qualityChecks: projectForm.value.qualityChecks,
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

// 面板上点任务要落到那条任务，不是只落到项目首页，否则还得在侧栏里再找一次。
const openTask = (task: CodeTaskListItem) => {
  router.push({ path: `/code/project/${task.projectId}`, query: { taskId: String(task.id) } })
}

const openNewProjectTask = (projectId: number) => {
  router.push({ path: `/code/project/${projectId}`, query: { newTask: '1' } })
}

const projectMenuOptions = computed(() =>
  projects.value.map(project => ({
    label: () => h(CodeProjectIdentity, { projectId: project.id, name: project.name }),
    key: `project-${project.id}`,
    children: [
      { label: t('code.enterProject'), key: `enter:${project.id}` },
      { label: t('code.quickPanel'), key: `panel:${project.id}` },
      { label: t('code.editProject'), key: `edit:${project.id}` },
    ],
  })),
)

const handleProjectMenuSelect = (key: string) => {
  const [action, rawId] = String(key).split(':')
  handleProjectAction(action, Number(rawId))
}

const handleProjectAction = (action: string, projectId: number) => {
  const project = projects.value.find(item => item.id === projectId)
  if (!project) return
  if (action === 'enter') enterProject(project.id)
  else if (action === 'panel') openQuickPanel(project)
  else if (action === 'edit') openEditProjectModal(project)
}

// 勾掉某个仓库即写入排除清单。主交付仓库在模板里已经禁用，这里再兜一层：
// 排除主仓会让交付无处落地，后端保存时也会拒绝。
const toggleRepositoryIncluded = (path: string, included: boolean) => {
  if (!included && path === projectForm.value.primaryRepository) return
  const excluded = new Set(projectForm.value.excludedRepositories)
  if (included) excluded.delete(path)
  else excluded.add(path)
  projectForm.value.excludedRepositories = [...excluded]
}

const openQuickPanel = (project: AIProject) => quickPanelsRef.value?.open(project)
</script>
