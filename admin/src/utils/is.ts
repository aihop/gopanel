export const isFail: (code: number) => boolean = (code: number) => code !== 0
export const isSucc: (code: number) => boolean = (code: number) => code === 0

/**
 * @description 判断是否是手机号
 * @param value
 * @returns {boolean}
 */
export function isMobile(value: string): boolean {
	const reg = /^1\d{10}$/
	return reg.test(value)
}
export function isEmail(email: string): boolean {
	const reg = /^[\w-]+(\.[\w-]+)*@[\w-]+(\.[\w-]+)+$/
	return reg.test(email)
}
export function isArray(val: any): val is Array<any> {
	return val && Array.isArray(val)
}
