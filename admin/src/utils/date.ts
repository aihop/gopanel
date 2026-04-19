export function handleDate(date: string) {
	return new Date(date).toLocaleDateString("en", { year: "numeric", month: "short", day: "numeric" })
}

export const formatTime = (timeStr: string): string => {
	if (!timeStr || timeStr === "0001-01-01T00:00:00Z") return "-"
	const date = new Date(timeStr)
	return date.toLocaleString("zh-CN")
}
