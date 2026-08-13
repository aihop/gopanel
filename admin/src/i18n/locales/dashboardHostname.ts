export const dashboardHostnameMessages = {
	zh: {
		dashboardHostname: {
			edit: "修改",
			save: "保存",
			cancel: "取消",
			placeholder: "请输入服务器名称",
			rule: "仅支持英文字母、数字、点和连字符；每段不能以连字符开头或结尾，最长 253 个字符",
			updated: "服务器名称已更新",
			updateFailed: "服务器名称修改失败",
			ErrHostnameInvalid: "服务器名称格式不正确",
			ErrHostnameToolUnavailable: "当前系统不支持修改服务器名称",
			ErrHostnameUpdateFailed: "服务器名称修改失败，请检查系统权限"
		}
	},
	en: {
		dashboardHostname: {
			edit: "Edit",
			save: "Save",
			cancel: "Cancel",
			placeholder: "Enter a server hostname",
			rule: "Use only letters, numbers, dots, and hyphens. Labels cannot start or end with a hyphen. Maximum length: 253 characters.",
			updated: "Server hostname updated",
			updateFailed: "Failed to update the server hostname",
			ErrHostnameInvalid: "The server hostname format is invalid",
			ErrHostnameToolUnavailable: "This system does not support hostname updates",
			ErrHostnameUpdateFailed: "Failed to update the server hostname. Check system permissions."
		}
	}
} as const
