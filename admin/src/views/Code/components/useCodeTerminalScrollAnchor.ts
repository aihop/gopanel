import { ref } from "vue"
import { isTerminalViewportAtBottom, type TerminalViewport } from "./codeTerminalSession"

/**
 * 「回到最新输出」按钮的状态。
 *
 * 终端实例是在 initTerminal 里才 new 出来的，所以这里收一个取值函数而不是实例本身，
 * 拿不到实例时按「贴底」算——没有内容的终端不该先挂一个箭头出来。
 */
export function useCodeTerminalScrollAnchor(getTerminal: () => TerminalViewport | null) {
	const scrolledUp = ref(false)

	const syncScrollAnchor = () => {
		const terminal = getTerminal()
		if (!terminal) {
			scrolledUp.value = false
			return
		}
		const { baseY, viewportY } = terminal.buffer.active
		scrolledUp.value = !isTerminalViewportAtBottom(baseY, viewportY)
	}

	const jumpToTerminalBottom = () => {
		getTerminal()?.scrollToBottom()
		syncScrollAnchor()
	}

	return { scrolledUp, syncScrollAnchor, jumpToTerminalBottom }
}
