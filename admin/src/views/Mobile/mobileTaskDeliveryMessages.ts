export const mobileTaskDeliveryMessages = {
	zh: {
		mobile: {
			saveChanges: "保存修改",
			checkingChanges: "检查修改",
			retryGitStatus: "重新检查",
			gitStatusFailed: "修改状态加载失败",
			gitSaveSuccess: "修改已保存，可继续交付",
			gitSaveFailed: "修改保存失败",
			deliverToMain: "交付到主仓",
			deliverToMainConfirm:
				"将已保存的代码进入统一交付队列，自动检查、合并并推送到主仓。",
			confirmDeliveryToMain: "确认交付",
			deliveryQueuedShort: "等待交付",
			deliveringShort: "交付中",
			runningDeliveryQuality: "正在运行质量检查",
			deliveredShort: "已交付",
			retryDelivery: "重新交付",
			deliveryQueuedSuccess: "任务已进入统一交付队列",
			deliveryQueueFailed: "任务交付失败"
		}
	},
	en: {
		mobile: {
			saveChanges: "Save changes",
			checkingChanges: "Checking changes",
			retryGitStatus: "Check again",
			gitStatusFailed: "Failed to load change status",
			gitSaveSuccess: "Changes saved and ready for delivery",
			gitSaveFailed: "Failed to save changes",
			deliverToMain: "Deliver to main",
			deliverToMainConfirm:
				"Queue saved code for checks, merge, push, and remote verification.",
			confirmDeliveryToMain: "Deliver",
			deliveryQueuedShort: "Queued",
			deliveringShort: "Delivering",
			runningDeliveryQuality: "Running quality checks",
			deliveredShort: "Delivered",
			retryDelivery: "Retry delivery",
			deliveryQueuedSuccess: "Task added to the delivery queue",
			deliveryQueueFailed: "Failed to queue task delivery"
		}
	}
} as const
