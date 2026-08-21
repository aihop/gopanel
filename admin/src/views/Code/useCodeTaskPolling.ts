import { watch } from "vue"
import type { ComputedRef, Ref } from "vue"
import { useDocumentVisibility, useIntervalFn } from "@vueuse/core"
import { getAITasks } from "@/api/modules/code"
import type { CodeTaskListItem } from "@/api/interface/codeTasks"

const ACTIVE_TASK_STATUSES = ["active", "running", "queued", "pending_approval", "delivering"]
type CodeTaskGitMode = "none" | "live" | "full"

export interface CodeTaskPollingOptions {
	/** 轮询间隔。开发面板是常驻页面，用比工作台更松的节奏。 */
	intervalMs?: number
	/** 每隔几轮带一次 git 汇总。git 汇总要按会话读工作区算 diff，不能每轮都拉。 */
	gitEveryPolls?: number
	limit?: number
	allPages?: boolean
	order?: "recent"
	/**
	 * 跨项目模式：projectId 传 0，后端按当前用户列出全部任务。
	 * 不开这个开关时 projectId 为 0 表示「项目还没选好」，直接跳过请求。
	 */
	allProjects?: boolean
	/** 只列归档任务。传 ref 是因为归档视图是可切换的，切换后下一轮就该换列表。 */
	archived?: Ref<boolean>
	/**
	 * 全部任务都空闲时改用的间隔。
	 * 没有任何东西在跑的时候还按秒级刷新纯属浪费；开了这个就自适应降频。
	 */
	idleIntervalMs?: number
	/** 自动 Git 刷新始终包含当前选中任务，即使任务已完成。 */
	selectedTaskId?: Ref<number | null>
}

export function useCodeTaskPolling(
	projectId: ComputedRef<number>,
	tasks: Ref<CodeTaskListItem[]>,
	taskTotal: Ref<number>,
	onError: (error: unknown) => void,
	options: CodeTaskPollingOptions = {}
) {
	const {
		intervalMs = 3000,
		gitEveryPolls = 10,
		limit = 50,
		allPages = false,
		order,
		allProjects = false,
		archived,
		idleIntervalMs = 0,
		selectedTaskId
	} = options
	let requestPending = false
	let activeGitMode: CodeTaskGitMode = "none"
	let activeSelectedTaskId = 0
	let pendingGitMode: CodeTaskGitMode = "none"
	let pollCount = 0
	const fetchTasks = async (silent = false, gitMode: CodeTaskGitMode = "full") => {
		if (!allProjects && !projectId.value) return
		const requestedSelectedTaskId = selectedTaskId?.value || 0
		if (requestPending) {
			if (gitMode === activeGitMode && (gitMode !== "live" || requestedSelectedTaskId === activeSelectedTaskId))
				return
			pendingGitMode = strongerGitMode(pendingGitMode, gitMode)
			return
		}
		const requestedProjectId = allProjects ? 0 : projectId.value
		const requestedArchived = archived?.value === true
		requestPending = true
		activeGitMode = gitMode
		activeSelectedTaskId = requestedSelectedTaskId
		try {
			const loaded = await fetchCodeTaskPages(
				async page => {
					const pageGitMode = page === 1 || gitMode === "none" ? gitMode : "live"
					const response = await getAITasks({
						page,
						limit,
						projectId: requestedProjectId,
						includeGit: pageGitMode !== "none",
						...(pageGitMode === "live"
							? { gitScope: "live" as const, selectedTaskId: requestedSelectedTaskId }
							: {}),
						...(requestedArchived ? { archived: 1 as const } : {}),
						...(order ? { order } : {})
					})
					if (response.code !== 0) throw new Error(response.message)
					return { items: response.data.items || [], total: response.data.total || 0, gitMode: pageGitMode }
				},
				limit,
				allPages
			)
			if (!allProjects && projectId.value !== requestedProjectId) return
			// 归档视图切换时会有一个在途请求，回来晚了会把新列表覆盖成旧的那一份。
			if (requestedArchived !== (archived?.value === true)) return
			const previousTasks = new Map(tasks.value.map(task => [task.id, task]))
			const freshGitTaskIds = new Set<number>()
			const loadedTasks = new Map<number, CodeTaskListItem>()
			for (const page of loaded.pages) {
				for (const task of page.items) {
					if (!loadedTasks.has(task.id)) loadedTasks.set(task.id, task)
					if (
						page.gitMode === "full" ||
						(page.gitMode === "live" && taskNeedsLiveGitSummary(task, requestedSelectedTaskId))
					) {
						freshGitTaskIds.add(task.id)
					}
				}
			}
			tasks.value = [...loadedTasks.values()].map(task => {
				if (freshGitTaskIds.has(task.id)) return task
				const previous = previousTasks.get(task.id)
				return previous ? { ...task, summary: { ...task.summary, ...pickCodeTaskGitSummary(previous) } } : task
			})
			taskTotal.value = loaded.total
		} catch (error) {
			if (!silent) onError(error)
		} finally {
			requestPending = false
			activeGitMode = "none"
			activeSelectedTaskId = 0
			if (pendingGitMode !== "none") {
				const nextGitMode = pendingGitMode
				pendingGitMode = "none"
				void fetchTasks(true, nextGitMode)
			}
		}
	}
	const fetchTasksFast = async (silent = false) => {
		await fetchTasks(silent, "none")
		setTimeout(() => void fetchTasks(true, "live"), 0)
	}
	// 自适应节奏，替代「不管有没有事都按秒刷」：
	//   1. 页面不可见（切标签页/最小化）→ 完全不发请求，回来时立刻补一次；
	//   2. 没有任何活跃任务 → 降到 idleIntervalMs。
	// 比上 WebSocket 便宜得多，而实际感知差别很小 —— 正在看的那条任务，
	// 它的终端 WebSocket 本来就是实时的，列表只是外围监控。
	const visibility = useDocumentVisibility()
	const hasActiveTask = () => tasks.value.some(task => ACTIVE_TASK_STATUSES.includes(task.status))
	let idleTicks = 0
	useIntervalFn(() => {
		if (visibility.value === "hidden") return
		if (!hasActiveTask() && idleIntervalMs > intervalMs) {
			idleTicks++
			if (idleTicks < Math.round(idleIntervalMs / intervalMs)) return
		}
		idleTicks = 0
		pollCount++
		void fetchTasks(true, pollCount % gitEveryPolls === 0 ? "live" : "none")
	}, intervalMs)

	// 切回页面立刻刷一次，不让用户对着一屏可能已经过时的数据等下一拍。
	watch(visibility, value => {
		if (value === "visible") void fetchTasks(true, "live")
	})
	if (selectedTaskId) {
		watch(selectedTaskId, (value, previous) => {
			if (value && value !== previous) void fetchTasks(true, "live")
		})
	}

	return { fetchTasks, fetchTasksFast }
}

interface CodeTaskPage<T> {
	items: T[]
	total: number
	gitMode: CodeTaskGitMode
}

export async function fetchCodeTaskPages<T>(
	loadPage: (page: number) => Promise<CodeTaskPage<T>>,
	limit: number,
	allPages: boolean
) {
	const firstPage = await loadPage(1)
	if (!allPages || firstPage.total <= firstPage.items.length) {
		return { pages: [firstPage], total: firstPage.total }
	}
	const pageCount = Math.ceil(firstPage.total / limit)
	const remainingPages = await Promise.all(Array.from({ length: pageCount - 1 }, (_, index) => loadPage(index + 2)))
	return { pages: [firstPage, ...remainingPages], total: firstPage.total }
}

function strongerGitMode(current: CodeTaskGitMode, requested: CodeTaskGitMode): CodeTaskGitMode {
	const priority = { none: 0, live: 1, full: 2 }
	return priority[requested] > priority[current] ? requested : current
}

function taskNeedsLiveGitSummary(task: CodeTaskListItem, selectedTaskId: number) {
	return task.id === selectedTaskId || ACTIVE_TASK_STATUSES.includes(task.status)
}

function pickCodeTaskGitSummary(task: CodeTaskListItem) {
	return {
		gitStatus: task.summary.gitStatus,
		gitError: task.summary.gitError,
		branch: task.summary.branch,
		repositories: task.summary.repositories,
		additions: task.summary.additions,
		deletions: task.summary.deletions,
		changedFiles: task.summary.changedFiles,
		hasDiff: task.summary.hasDiff,
		unsavedAdditions: task.summary.unsavedAdditions,
		unsavedDeletions: task.summary.unsavedDeletions,
		unsavedFiles: task.summary.unsavedFiles,
		hasUnsavedChanges: task.summary.hasUnsavedChanges
	}
}
