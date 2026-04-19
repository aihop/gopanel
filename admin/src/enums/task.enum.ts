export enum CardTaskType {
	BatchCard = "batchcard", // 批量开卡
	SingleCard = "singlecard" // 单卡开卡
}

export enum CardTaskStatus {
	Processing = 10, // 执行中
	Completed = 20, // 处理完成
	Failed = 30 // 处理失败
}
