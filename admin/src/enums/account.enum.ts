export enum AccountRecordStatus {
	Unknown = 0,
	Pending = 10, // 待审核
	Approved = 20, // 审核通过
	Rejected = 30 // 审核拒绝
}

export const AccountRecordStatusDesc = {
	[AccountRecordStatus.Unknown]: "未知状态",
	[AccountRecordStatus.Pending]: "待审核",
	[AccountRecordStatus.Approved]: "审核通过",
	[AccountRecordStatus.Rejected]: "审核拒绝"
}
