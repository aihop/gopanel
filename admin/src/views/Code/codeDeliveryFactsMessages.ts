import { codeGitReviewMessages } from "./codeGitReviewMessages"

export const codeDeliveryFactsMessages = {
	zh: {
		code: {
			...codeGitReviewMessages.zh.code,
			gitDeliveryDetailsShow: "交付详情",
			gitDeliveryDetailsHide: "收起详情",
			gitDeliveryFactStatus_mergeConflict: "合并时发生冲突",
			gitDeliveryFactStatus_stoppedAfterConflict: "冲突后未继续"
		}
	},
	en: {
		code: {
			...codeGitReviewMessages.en.code,
			gitDeliveryDetailsShow: "Delivery details",
			gitDeliveryDetailsHide: "Hide details",
			gitDeliveryFactStatus_mergeConflict: "Conflict occurred during merge",
			gitDeliveryFactStatus_stoppedAfterConflict: "Stopped after the conflict"
		}
	}
} as const
