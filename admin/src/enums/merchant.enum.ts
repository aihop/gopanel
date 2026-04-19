export enum MerchantRecordStatus {
	Unknown = 0,
	Pending = 10, // 待审核
	Approved = 20, // 审核通过
	Rejected = 30, // 审核拒绝
	Frozen = 40
}

export const MerchantRecordStatusDesc = {
	[MerchantRecordStatus.Unknown]: "未知状态",
	[MerchantRecordStatus.Pending]: "待审核",
	[MerchantRecordStatus.Approved]: "审核通过",
	[MerchantRecordStatus.Rejected]: "审核拒绝",
	[MerchantRecordStatus.Frozen]: "已冻结"
}
