const markdownTags = new Set([
	"a",
	"blockquote",
	"br",
	"code",
	"del",
	"div",
	"em",
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6",
	"hr",
	"img",
	"li",
	"ol",
	"p",
	"pre",
	"span",
	"strong",
	"table",
	"tbody",
	"td",
	"th",
	"thead",
	"tr",
	"ul",
])

const markdownAttributes = new Set(["alt", "href", "src", "title"])

export function sanitizeConversationMarkdown(html: string) {
	const documentNode = new DOMParser().parseFromString(html, "text/html")
	documentNode.body.querySelectorAll("*").forEach(element => {
		if (!markdownTags.has(element.tagName.toLowerCase())) {
			element.replaceWith(...Array.from(element.childNodes))
			return
		}
		for (const attribute of Array.from(element.attributes)) {
			const name = attribute.name.toLowerCase()
			if (!markdownAttributes.has(name)) {
				element.removeAttribute(attribute.name)
				continue
			}
			if (name === "href" && !/^(https?:|mailto:|tel:|\/|\.\/|\.\.\/|#)/i.test(attribute.value.trim())) {
				element.removeAttribute(attribute.name)
			}
			if (name === "src" && !/^(https?:|data:image\/|\/|\.\/|\.\.\/)/i.test(attribute.value.trim())) {
				element.removeAttribute(attribute.name)
			}
		}
		if (element.tagName.toLowerCase() === "a") element.setAttribute("rel", "noopener noreferrer")
	})
	return documentNode.body.innerHTML
}

export function isConversationSubmitKey(event: { key: string; shiftKey: boolean; isComposing?: boolean; keyCode?: number }) {
	return event.key === "Enter" && !event.shiftKey && !event.isComposing && event.keyCode !== 229
}
