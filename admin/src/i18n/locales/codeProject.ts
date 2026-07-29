export const codeProjectMessages = {
	zh: {
		code: {
			createProject: "新建项目",
			noProject: "暂无项目，请先新建一个",
			projectName: "项目名称",
			projectNameRequired: "请输入项目名称",
			projectDesc: "项目描述（可选）",
			projectDirectory: "项目路径",
			projectDirectoryRequired: "请选择项目路径",
			selectProjectDirectory: "选择项目目录",
			browseDirectory: "选择目录",
			parentDirectory: "上一级",
			refreshDirectory: "刷新",
			currentDirectory: "当前目录",
			selectThisDirectory: "选择当前目录",
			noSubdirectories: "当前目录没有子目录，可以直接选择当前目录",
			directoryLoadFailed: "目录加载失败，请重试",
			projectLoadFailed: "项目列表加载失败，请重试",
			projectCreateSuccess: "项目创建成功",
			projectCreateFailed: "项目创建失败",
			projectPath: "项目路径",
			project: "项目",
			projectFallback: "项目",
			noProjectHistory: "该项目暂无对话历史",
			enterProject: "进入项目",
			sessionUsesProjectDirectory: "本次会话将使用项目绑定的工作目录。"
		}
	},
	en: {
		code: {
			createProject: "Create Project",
			noProject: "No projects yet. Create one to get started.",
			projectName: "Project name",
			projectNameRequired: "Enter a project name",
			projectDesc: "Project description (optional)",
			projectDirectory: "Project directory",
			projectDirectoryRequired: "Select a project directory",
			selectProjectDirectory: "Select Project Directory",
			browseDirectory: "Browse",
			parentDirectory: "Parent",
			refreshDirectory: "Refresh",
			currentDirectory: "Current directory",
			selectThisDirectory: "Select current directory",
			noSubdirectories: "This directory has no subdirectories. You can select the current directory.",
			directoryLoadFailed: "Failed to load directories. Please retry.",
			projectLoadFailed: "Failed to load projects. Please retry.",
			projectCreateSuccess: "Project created",
			projectCreateFailed: "Failed to create project",
			projectPath: "Project path",
			project: "Project",
			projectFallback: "Project",
			noProjectHistory: "No conversation history for this project",
			enterProject: "Open Project",
			sessionUsesProjectDirectory: "This session uses the working directory configured for the project."
		}
	}
} as const
