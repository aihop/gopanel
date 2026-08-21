import { computed, nextTick, onBeforeUnmount, ref, watch, type Ref } from "vue"

/**
 * 判定「贴底」的容差。子像素和缩放会让差值停在 1px 上下，
 * 严格相等会把「正在跟看最新消息」误判成「用户翻到历史里去了」。
 */
const BOTTOM_TOLERANCE = 24

/** 量不到输入框高度时的兜底留白，对应原先写死的 pb-40。 */
const FALLBACK_COMPOSER_SPACE = 160

/** 输入框和最后一条消息之间的呼吸空间。 */
const COMPOSER_GAP = 24

/**
 * 视口是不是贴着底部。容差不能省：子像素和缩放会让差值停在 1px 上下，
 * 严格相等会把「正在跟看最新消息」误判成「用户翻上去了」，跟随就此断掉。
 */
export function isConversationAtBottom(scrollHeight: number, scrollTop: number, clientHeight: number) {
	return scrollHeight - scrollTop - clientHeight <= BOTTOM_TOLERANCE
}

/**
 * 列表底部要留多少空白。输入框是绝对定位浮在列表上的，原先用写死的 pb-40 让位，
 * 但输入框会随输入内容长高——超过那个固定值时最后一条消息就被压在下面，
 * 即使真的滚到底也像是没到底。
 */
export function conversationListPaddingBottom(composerHeight: number) {
	return `${(composerHeight || FALLBACK_COMPOSER_SPACE) + COMPOSER_GAP}px`
}

/**
 * 让对话列表稳定停在底部，并给浮在上面的输入框让出正确的留白。
 *
 * 只在 nextTick 里设一次 scrollTop 是不够的：消息用的 Markdown 渲染器是
 * defineAsyncComponent + 动态 import，nextTick 时那个 chunk 还没加载，列表几乎是空的，
 * 这时 scrollHeight 只有一屏高，滚了等于没滚；等 chunk 落地、Markdown 解析和代码高亮
 * 跑完，高度才暴涨，而没有任何人再滚第二次，于是停在顶部。
 *
 * 所以改成「跟随」而不是「滚一次」：只要用户没有主动往上翻，内容高度每变一次就重新贴底。
 * 异步 Markdown、图片加载、流式增量输出，全部由同一个机制兜住。
 */
export function useConversationScrollAnchor(
	listRef: Ref<HTMLElement | null>,
	contentRef: Ref<HTMLElement | null>,
	composerRef: Ref<HTMLElement | null>,
) {
	const followBottom = ref(true)
	const composerHeight = ref(0)
	let contentObserver: ResizeObserver | null = null
	let composerObserver: ResizeObserver | null = null

	const pinToBottom = () => {
		const list = listRef.value
		if (list) list.scrollTop = list.scrollHeight
	}

	/** 强制贴底：打开会话、自己发出消息这类「理应看最新」的时刻用。 */
	const scrollToBottom = async () => {
		followBottom.value = true
		await nextTick()
		pinToBottom()
	}

	/**
	 * 用户往上翻就停止跟随，翻回底部自动恢复。
	 * 没有这一条的话，流式输出每来一个增量都会把正在读历史的人拽回去。
	 */
	const handleScroll = () => {
		const list = listRef.value
		if (!list) return
		followBottom.value = isConversationAtBottom(list.scrollHeight, list.scrollTop, list.clientHeight)
	}

	const listPaddingBottom = computed(() => conversationListPaddingBottom(composerHeight.value))

	const observe = (element: HTMLElement | null, onResize: () => void) => {
		if (!element || typeof ResizeObserver === "undefined") return null
		const observer = new ResizeObserver(onResize)
		observer.observe(element)
		return observer
	}

	watch(
		contentRef,
		content => {
			contentObserver?.disconnect()
			contentObserver = observe(content, () => {
				if (followBottom.value) pinToBottom()
			})
		},
		{ immediate: true },
	)

	watch(
		composerRef,
		composer => {
			composerObserver?.disconnect()
			composerHeight.value = composer?.offsetHeight || 0
			composerObserver = observe(composer, () => {
				composerHeight.value = composer?.offsetHeight || 0
				// 输入框变高会遮住底部内容，跟随状态下要把被吃掉的部分补回来。
				if (followBottom.value) void nextTick(pinToBottom)
			})
		},
		{ immediate: true },
	)

	onBeforeUnmount(() => {
		contentObserver?.disconnect()
		composerObserver?.disconnect()
	})

	return { followBottom, scrollToBottom, handleScroll, listPaddingBottom }
}
