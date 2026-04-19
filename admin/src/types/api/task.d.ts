import type { CardTaskStatus, CardTaskType } from "@/enums/task.enum"

export interface CardTask {
	id: number
	types: CardTaskType // 操作类型
	card_num: number // 卡片数量
	progress: number // 进度
	status: CardTaskStatus // 状态
	created_at: string
	updated_at: string
}
