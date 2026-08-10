export interface TerminalBufferLineReader {
	readonly isWrapped: boolean
	translateToString(trimRight?: boolean, startColumn?: number, endColumn?: number): string
}

export interface TerminalBufferReader {
	readonly length: number
	getLine(index: number): TerminalBufferLineReader | undefined
}

export function terminalBufferText(buffer: TerminalBufferReader): string {
	let output = ""
	let hasLine = false
	for (let index = 0; index < buffer.length; index++) {
		const line = buffer.getLine(index)
		if (!line) continue
		if (hasLine && !line.isWrapped) output += "\n"
		output += line.translateToString(true)
		hasLine = true
	}
	return output.replace(/\n+$/, "")
}
