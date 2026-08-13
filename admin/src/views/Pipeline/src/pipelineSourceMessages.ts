export const pipelineSourceMessages = {
	zh: {
		pipelineSource: {
			sourceType: "代码来源",
			sourceGit: "Git 仓库（默认）",
			sourceCode: "开发项目",
			codeProject: "开发项目",
			codeProjectPlaceholder: "选择要快照构建的开发项目",
			codeProjectHelper: "执行时会冻结并复制项目当前内容，不会从远程 Git 拉取；.env、Git 元数据和依赖缓存不会进入快照。",
			codeProjectRequired: "请选择开发项目",
			codeProjectsLoadFailed: "开发项目加载失败，请关闭后重试",
			retry: "重试",
			codeProjectsEmpty: "暂无可用开发项目，请先在开发中心创建项目",
			repoUrl: "仓库地址",
			authType: "认证方式",
			authData: "凭证信息",
			authDataPlaceholder: "填写 Token 或 Password",
			recordSource: "代码来源",
			recordSourceGit: "Git",
			recordSourceCode: "Code #{id}"
		}
	},
	en: {
		pipelineSource: {
			sourceType: "Code source",
			sourceGit: "Git repository (default)",
			sourceCode: "Code project",
			codeProject: "Code project",
			codeProjectPlaceholder: "Select a Code project to snapshot and build",
			codeProjectHelper: "Each run freezes and copies the current project content without pulling remote Git. .env files, Git metadata, and dependency caches are excluded.",
			codeProjectRequired: "Select a Code project",
			codeProjectsLoadFailed: "Failed to load Code projects. Close this dialog and retry.",
			retry: "Retry",
			codeProjectsEmpty: "No Code projects are available. Create one in Code first.",
			repoUrl: "Repository URL",
			authType: "Authentication",
			authData: "Credentials",
			authDataPlaceholder: "Enter a token or password",
			recordSource: "Code source",
			recordSourceGit: "Git",
			recordSourceCode: "Code #{id}"
		}
	}
}
