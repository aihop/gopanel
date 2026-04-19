function formatCurrency(
	value: number | string | undefined | null,
	currency: string,
	options?: Intl.NumberFormatOptions
): string {
	const numberOptions = {
		minimumFractionDigits: 2,
		maximumFractionDigits: 2,
		...options
	}
	const numValue = Number(value)
	if (Number.isNaN(numValue)) {
		return typeof value === "string" ? value : ""
	}
	switch (currency) {
		case "USD":
			return new Intl.NumberFormat("en-US", { style: "currency", currency: "USD", ...numberOptions }).format(
				numValue
			)
		case "CNY":
		case "RMB":
			return new Intl.NumberFormat("zh-CN", { style: "currency", currency: "CNY", ...numberOptions }).format(
				numValue
			)
		default:
			return new Intl.NumberFormat("en-US", { ...numberOptions }).format(numValue)
	}
}

function getCurrencySymbol(currency?: string): string {
	switch (currency) {
		case "USD":
			return "$"
		case "CNY":
		case "RMB":
			return "¥"
		default:
			return ""
	}
}

export { formatCurrency, getCurrencySymbol }
