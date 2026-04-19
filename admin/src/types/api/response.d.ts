export interface Response<T> {
	code: number
	msg: string
	data: T
}

export interface Pagination<T> {
	currentPage: number
	data: T[]
	total: number
}

export interface Options {
	label: string
	value: string | number
}
