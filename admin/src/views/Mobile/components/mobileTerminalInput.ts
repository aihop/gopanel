export function insertTerminalSymbol(draft: string, symbol: string, start: number, end: number) {
	const selectionStart = Math.max(0, Math.min(start, draft.length))
	const selectionEnd = Math.max(selectionStart, Math.min(end, draft.length))
	return {
		value: `${draft.slice(0, selectionStart)}${symbol}${draft.slice(selectionEnd)}`,
		cursor: selectionStart + symbol.length
	}
}
