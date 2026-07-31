export const mobileRepositorySyncMessages = {
	zh: {
		mobileSync: {
			title: "本地主仓同步",
			action: "同步主仓",
			confirm: "仅在主仓干净且可以快进时同步，不会覆盖本地提交。确认继续？",
			cancel: "取消",
			success: "本地主仓已安全同步",
			failed: "本地主仓同步失败",
			loadFailed: "主仓同步状态加载失败",
			retry: "重试",
			summary: "{count} 个仓库 · 本地领先 {ahead} · 远端领先 {behind}",
			status_synced: "已同步",
			status_local: "仅本地",
			status_behind: "可同步",
			status_ahead: "本地领先",
			status_diverged: "已分叉",
			status_dirty: "有未提交变更",
			status_offline: "远端不可用",
			status_blocked: "同步被占用"
		}
	},
	en: {
		mobileSync: {
			title: "Local repository sync",
			action: "Sync",
			confirm:
				"Sync only clean repositories that can fast-forward. Local commits will not be overwritten. Continue?",
			cancel: "Cancel",
			success: "Local repositories synced safely",
			failed: "Failed to sync local repositories",
			loadFailed: "Failed to load repository sync status",
			retry: "Retry",
			summary: "{count} repositories · {ahead} ahead · {behind} behind",
			status_synced: "Synced",
			status_local: "Local only",
			status_behind: "Ready to sync",
			status_ahead: "Local ahead",
			status_diverged: "Diverged",
			status_dirty: "Uncommitted changes",
			status_offline: "Remote unavailable",
			status_blocked: "Sync blocked"
		}
	}
} as const
