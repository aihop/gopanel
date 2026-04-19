export enum domainStatusEnum {
	Expired = 10,
	Effective = 20,
	Expiration = 30
}

export const domainStatusText = {
	[domainStatusEnum.Expired]: "已到期",
	[domainStatusEnum.Effective]: "已支付",
	[domainStatusEnum.Expiration]: "即将到期"
}
