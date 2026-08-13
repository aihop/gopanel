export const systemDiagnosticMessages = {
	zh: {
		systemDiagnostic: {
			title: "智能诊断",
			open: "打开智能诊断",
			close: "关闭",
			loading: "正在读取 GoPanel 状态",
			loadFailed: "诊断中心加载失败，请重试",
			retry: "重试",
			readOnly: "只读诊断",
			healthy: "控制面正常",
			unhealthy: "控制面异常",
			noAccount: "没有可用的 AI 账号，请先在 Code 中配置并启用一个模型账号。",
			selectAI: "选择执行诊断的 AI",
			emptyTitle: "可以直接询问 GoPanel 当前状态",
			emptyDescription: "AI 可读取脱敏后的系统快照，并通过受控只读 SQL 查询面板数据库。",
			placeholder: "例如：为什么最近一次数据库备份失败？",
			send: "诊断",
			sending: "正在分析",
			sendFailed: "诊断失败，请检查 AI 账号或稍后重试",
			inputRequired: "请输入诊断问题",
			quickBackup: "分析最近的数据库备份失败",
			quickControlPlane: "检查 gpc 和 gp-agent 是否正常",
			quickFailures: "汇总最近的系统失败并排序",
			privacy: "不会向模型提供密码、Token、API Key 或数据库凭据；数据修正必须另行审批。",
			drag: "上下拖动工具栏"
		}
	},
	en: {
		systemDiagnostic: {
			title: "AI Diagnostics",
			open: "Open AI diagnostics",
			close: "Close",
			loading: "Reading GoPanel status",
			loadFailed: "Failed to load diagnostics. Please retry.",
			retry: "Retry",
			readOnly: "Read-only",
			healthy: "Control plane healthy",
			unhealthy: "Control plane unhealthy",
			noAccount: "No AI account is available. Configure and enable one in Code first.",
			selectAI: "Select the AI for diagnostics",
			emptyTitle: "Ask about the current GoPanel state",
			emptyDescription: "The AI can read a scrubbed system snapshot and query panel data through controlled read-only SQL.",
			placeholder: "For example: Why did the latest database backup fail?",
			send: "Diagnose",
			sending: "Analyzing",
			sendFailed: "Diagnostics failed. Check the AI account or retry later.",
			inputRequired: "Enter a diagnostic question",
			quickBackup: "Analyze recent database backup failures",
			quickControlPlane: "Check gpc and gp-agent health",
			quickFailures: "Summarize and rank recent system failures",
			privacy: "Passwords, tokens, API keys, and database credentials are never sent to the model. Data changes require separate approval.",
			drag: "Drag the toolbar vertically"
		}
	}
} as const
