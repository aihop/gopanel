import { computed, ref } from "vue"
import { useMessage } from "naive-ui"
import { useI18n } from "vue-i18n"
import { createAIProject, discoverCodeProjectRepositories, updateAIProject } from "@/api/modules/code"
import type { AIProject, CodeProjectQualityCheck } from "@/api/interface/code"
import { codeProjectMessages } from "@/i18n/locales/codeProject"

export function useCodeProjectForm() {
	const message = useMessage()
	const { t } = useI18n({ messages: codeProjectMessages })

	const showCreateProjectModal = ref(false)
	const showDirectoryPicker = ref(false)
	const creatingProject = ref(false)
	const repositoriesLoading = ref(false)
	const repositoryOptions = ref<Array<{ label: string; value: string }>>([])
	const editingProjectId = ref<number | null>(null)
	const defaultWorkDir = ref("/")
	const directoryRoot = ref("/")

	const emptyQualityChecks = (): CodeProjectQualityCheck[] => []
	const projectForm = ref({
		name: "",
		desc: "",
		workDir: "",
		sourceDirs: [] as string[],
		excludedRepositories: [] as string[],
		primaryRepository: "",
		deliveryBranch: "",
		deliveryMode: "direct" as "direct" | "branch",
		gitCredentialId: 0,
		requireQualityGate: true,
		qualityChecks: emptyQualityChecks(),
		monthlyTokenBudget: 0,
	})

	const primaryRepositoryOptions = computed(() => [
		{ label: t("code.primaryRepositoryAuto"), value: "" },
		...repositoryOptions.value,
	])

	const deliveryModeOptions = computed(() => [
		{ label: t("code.deliveryModeDirect"), value: "direct" },
		{ label: t("code.deliveryModeBranch"), value: "branch" },
	])

	const loadRepositoryOptions = async () => {
		const sourceDirs = projectForm.value.sourceDirs
		if (!sourceDirs.length) {
			repositoryOptions.value = []
			projectForm.value.primaryRepository = ""
			return
		}
		repositoriesLoading.value = true
		try {
			const res = await discoverCodeProjectRepositories(sourceDirs)
			repositoryOptions.value = (res.data || []).map(item => ({
				label: `${item.name} · ${item.path}`,
				value: item.path,
			}))
			if (
				projectForm.value.primaryRepository &&
				!repositoryOptions.value.some(item => item.value === projectForm.value.primaryRepository)
			) {
				projectForm.value.primaryRepository = ""
			}
		} catch {
			repositoryOptions.value = []
			projectForm.value.primaryRepository = ""
		} finally {
			repositoriesLoading.value = false
		}
	}

	const openCreateProjectModal = () => {
		editingProjectId.value = null
		projectForm.value = {
			name: "",
			desc: "",
			workDir: defaultWorkDir.value,
			sourceDirs: [],
			excludedRepositories: [],
			primaryRepository: "",
			deliveryBranch: "",
			deliveryMode: "direct",
			gitCredentialId: 0,
			requireQualityGate: true,
			qualityChecks: emptyQualityChecks(),
			monthlyTokenBudget: 0,
		}
		showCreateProjectModal.value = true
		repositoryOptions.value = []
	}

	const openEditProjectModal = (project: AIProject) => {
		editingProjectId.value = project.id
		const sourceDirs = project.sourceDirs?.length ? project.sourceDirs : project.workDir ? [project.workDir] : []
		projectForm.value = {
			name: project.name,
			desc: project.description || "",
			workDir: sourceDirs[0] || defaultWorkDir.value,
			sourceDirs,
			excludedRepositories: [...(project.excludedRepositories || [])],
			primaryRepository: project.primaryRepository || "",
			deliveryBranch: project.deliveryBranch || "main",
			deliveryMode: project.deliveryMode === "branch" ? "branch" : "direct",
			gitCredentialId: project.gitCredentialId || 0,
			requireQualityGate: project.requireQualityGate,
			qualityChecks: (project.qualityChecks || []).map(check => ({ ...check })),
			monthlyTokenBudget: project.monthlyTokenBudget || 0,
		}
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

	const toggleRepositoryIncluded = (path: string, included: boolean) => {
		if (!included && path === projectForm.value.primaryRepository) return
		const excluded = new Set(projectForm.value.excludedRepositories)
		if (included) excluded.delete(path)
		else excluded.add(path)
		projectForm.value.excludedRepositories = [...excluded]
	}

	const submitProject = async () => {
		if (!projectForm.value.name.trim()) {
			message.warning(t("code.projectNameRequired"))
			return
		}
		if (projectForm.value.sourceDirs.length === 0) {
			message.warning(t("code.projectDirectoryRequired"))
			return
		}
		creatingProject.value = true
		try {
			const payload = {
				name: projectForm.value.name.trim(),
				description: projectForm.value.desc.trim(),
				sourceDirs: projectForm.value.sourceDirs,
				excludedRepositories: projectForm.value.excludedRepositories,
				primaryRepository: projectForm.value.primaryRepository?.trim() || "",
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
				message.success(t(editingProjectId.value ? "code.projectUpdateSuccess" : "code.projectCreateSuccess"))
				return true
			}
		} catch {
			void 0
		} finally {
			creatingProject.value = false
		}
		return false
	}

	return {
		showCreateProjectModal,
		showDirectoryPicker,
		creatingProject,
		repositoriesLoading,
		repositoryOptions,
		editingProjectId,
		projectForm,
		defaultWorkDir,
		directoryRoot,
		primaryRepositoryOptions,
		deliveryModeOptions,
		openCreateProjectModal,
		openEditProjectModal,
		removeSourceDir,
		handleSourceDirsSelected,
		toggleRepositoryIncluded,
		submitProject,
	}
}