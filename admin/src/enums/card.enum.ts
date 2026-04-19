export enum CardStatus {
	Unknown = 0,
	Pending = 10, // 待处理 - 新卡申请中或正在制卡
	Inactive = 20, // 未激活 - 已创建但未激活
	Active = 30, // 已激活 - 正常可用状态
	Lost = 40, // 丢失 - 卡片丢失
	Stolen = 50, // 被盗 - 卡片被盗
	Closed = 60, // 已注销 - 卡片已被关闭
	Blocked = 70, // 已冻结 - 卡片被临时冻结
	Expired = 80, // 已过期 - 卡片已过有效期
	Failed = 90 // 操作失败 - 卡片创建/激活失败
}

export const CardStatusDesc = {
	[CardStatus.Unknown]: "未知状态",
	[CardStatus.Pending]: "待处理",
	[CardStatus.Inactive]: "未激活",
	[CardStatus.Active]: "已激活",
	[CardStatus.Lost]: "丢失",
	[CardStatus.Stolen]: "被盗",
	[CardStatus.Closed]: "已注销",
	[CardStatus.Blocked]: "已冻结",
	[CardStatus.Expired]: "已过期",
	[CardStatus.Failed]: "操作失败"
}

// 卡片状态辅助函数
export const CardStatusHelper = {
	isActive: (status: CardStatus): boolean => {
		return status === CardStatus.Active
	},
	isUsable: (status: CardStatus): boolean => {
		return [CardStatus.Active, CardStatus.Pending, CardStatus.Inactive].includes(status)
	}
}

export enum OpenCardStatus {
	Unopened = 10, // 未开卡
	Opened = 20, // 已开卡
	Activated = 30 // 已激活
}

export const OpenCardStatusDesc = {
	[OpenCardStatus.Unopened]: "未开卡",
	[OpenCardStatus.Opened]: "已开卡",
	[OpenCardStatus.Activated]: "已激活"
}

// 开卡状态辅助函数
export const OpenCardStatusHelper = {
	isOpened: (status: OpenCardStatus): boolean => {
		return [OpenCardStatus.Opened, OpenCardStatus.Activated].includes(status)
	},
	isActivated: (status: OpenCardStatus): boolean => {
		return status === OpenCardStatus.Activated
	}
}

// 卡片风控状态
export enum CardRiskStatus {
	Normal = 10, // 未风控 - 正常状态
	Risky = 20 // 已风控 - 被风控系统检测为高风险
}

export const CardRiskStatusDesc = {
	[CardRiskStatus.Normal]: "未风控",
	[CardRiskStatus.Risky]: "已风控"
}

// 风控状态辅助函数
export const CardRiskStatusHelper = {
	isRisky: (status: CardRiskStatus): boolean => {
		return status === CardRiskStatus.Risky
	},
	isNormal: (status: CardRiskStatus): boolean => {
		return status === CardRiskStatus.Normal
	}
}

// 卡片类型
export enum CardType {
	Share = "share", // 共享卡
	Recharge = "recharge" // 储值卡
}

export const CardTypeDesc = {
	[CardType.Share]: "共享卡",
	[CardType.Recharge]: "储值卡"
}
