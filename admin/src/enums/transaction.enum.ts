export enum TransactionType {
	UNKNOWN = "UNKNOWN",
	AUTHORIZATION = "AUTHORIZATION", // 消费授权 - 确定卡的有效性并预留资金
	CLEARING = "CLEARING", // 清算 - 完成之前批准的授权交易的结算
	REVERSAL = "REVERSAL", // 撤销/冲正 - 撤销之前批准的授权、清算或退款的交易
	REFUND = "REFUND", // 退款 - 针对之前已清算交易的退款请求
	FUND_IN = "FUND_IN", // 资金汇入 - 向卡内汇入资金
	CORRECTION = "CORRECTION", // 交易校正 - 纠正之前的授权或退款交易
	VERIFICATION = "VERIFICATION", // 验证 - 验证卡信息但不实际授权
	SERVICE_FEE = "SERVICE_FEE" // 服务费 - 各类服务费用收取
}

export enum TransactionStatus {
	UNKNOWN = 0,
	PENDING = 10, // 待处理 - 'pending', 'PENDING'
	PROCESSING = 20, // 处理中 - 'processing', 'APPROVED', 'authorized'
	SUCCESS = 30, // 已完成 - 'succeed', 'CLEARED'
	FAIL = 40, // 交易失败 - 'failed', 'FAILED', 'EXPIRED'
	REVERSED = 50 // 已撤销/作废 - 'void', 'REVERSED'
}

export const TransactionTypeDesc = {
	[TransactionType.UNKNOWN]: "未知状态",
	[TransactionType.AUTHORIZATION]: "消费授权",
	[TransactionType.CLEARING]: "清算",
	[TransactionType.REVERSAL]: "撤销/冲正",
	[TransactionType.REFUND]: "退款",
	[TransactionType.FUND_IN]: "资金汇入",
	[TransactionType.CORRECTION]: "交易校正",
	[TransactionType.VERIFICATION]: "验证",
	[TransactionType.SERVICE_FEE]: "服务费"
}

export const TransactionStatusDesc = {
	[TransactionStatus.UNKNOWN]: "未知状态",
	[TransactionStatus.PENDING]: "待处理",
	[TransactionStatus.PROCESSING]: "处理中",
	[TransactionStatus.SUCCESS]: "已完成",
	[TransactionStatus.FAIL]: "交易失败",
	[TransactionStatus.REVERSED]: "已撤销/作废"
}
