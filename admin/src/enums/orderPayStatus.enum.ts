// 付款
export enum orderPayStatusEnum {
	Unpaid = 10,
	Paid = 20
}

export const orderPayStatusText = {
	[orderPayStatusEnum.Unpaid]: "未支付",
	[orderPayStatusEnum.Paid]: "已支付"
}

export const orderPayStatusColor = {
	[orderPayStatusEnum.Unpaid]: "gray",
	[orderPayStatusEnum.Paid]: "orange"
}
