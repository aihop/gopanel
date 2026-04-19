export function getEnumDesc(enumObj: any, value: number | string, defaultValue: string = "未知"): string {
	return enumObj[value] || defaultValue
}
