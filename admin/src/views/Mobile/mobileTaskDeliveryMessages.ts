export const mobileTaskDeliveryMessages = {
	zh: {
		mobile: {
			deliverToMain: "交付到主仓",
			deliverToMainConfirm:
				"将已保存提交的代码进入统一交付队列，自动合并到主仓、检查并推送。未提交修改不会被交付，请先保存提交。",
			confirmDeliveryToMain: "确认交付",
			deliveringShort: "交付中",
			deliveredShort: "已交付",
			deliveryQueuedSuccess: "任务已进入统一交付队列",
			deliveryQueueFailed: "任务交付失败"
		}
	},
	en: {
		mobile: {
			deliverToMain: "Deliver to main",
			deliverToMainConfirm:
				"Queue saved commits for merge, checks, push, and remote verification. Uncommitted changes are not delivered; commit them first.",
			confirmDeliveryToMain: "Deliver",
			deliveringShort: "Delivering",
			deliveredShort: "Delivered",
			deliveryQueuedSuccess: "Task added to the delivery queue",
			deliveryQueueFailed: "Failed to queue task delivery"
		}
	}
} as const
