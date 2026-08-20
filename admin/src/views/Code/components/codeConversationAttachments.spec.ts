import { describe, expect, it, vi } from "vitest"
import {
	GOPANEL_PATH_MIME,
	attachmentsFromPaths,
	conversationAttachmentKind,
	decodeFileUri,
	extractDroppedAttachments,
	isWorkspaceRelativePath,
	parseConversationAttachments,
	serializeInstructionContent,
	toWorkspaceRelativePath,
	writeStructureDragData,
} from "./codeConversationAttachments"

class FakeTransfer {
	files: File[] = []
	private data: Record<string, string> = {}
	effectAllowed = "none"
	setData(type: string, value: string) {
		this.data[type] = value
	}
	getData(type: string) {
		return this.data[type] || ""
	}
}

describe("conversation attachment paths", () => {
	it("识别图片并保留工作区相对路径", () => {
		expect(conversationAttachmentKind("shot.PNG")).toBe("image")
		expect(conversationAttachmentKind("main.go")).toBe("file")
		expect(isWorkspaceRelativePath("src/main.go")).toBe(true)
		expect(isWorkspaceRelativePath("/tmp/shot.png")).toBe(false)
		expect(toWorkspaceRelativePath("/tmp/app/src/main.go", "/tmp/app")).toBe("src/main.go")
		expect(decodeFileUri("file:///Users/me/shot.png")).toBe("/Users/me/shot.png")
	})

	it("把附件写成 @attach 路径，展示时再拆开", () => {
		const content = serializeInstructionContent("看这张图", [{ path: "docs/login.png", name: "login.png", kind: "image" }])
		expect(content).toBe("看这张图\n\n@attach docs/login.png")
		expect(parseConversationAttachments(content)).toEqual({
			text: "看这张图",
			attachments: [{ path: "docs/login.png", name: "login.png", kind: "image" }],
		})
	})

	it("从项目结构拖拽数据里取出路径", () => {
		const transfer = new FakeTransfer()
		writeStructureDragData(transfer as unknown as DataTransfer, "admin/src/App.vue")
		expect(transfer.getData(GOPANEL_PATH_MIME)).toBe("admin/src/App.vue")
		expect(extractDroppedAttachments(transfer as unknown as DataTransfer).attachments).toEqual([
			{ id: "admin/src/App.vue", path: "admin/src/App.vue", name: "App.vue", kind: "file" },
		])
	})

	it("从 file:// 和带 path 的 File 取出附件", () => {
		const transfer = new FakeTransfer()
		transfer.setData("text/uri-list", "file:///tmp/app/docs/shot.png")
		const file = new File(["x"], "shot.png", { type: "image/png" })
		Object.defineProperty(file, "path", { value: "/tmp/app/docs/shot.png" })
		vi.stubGlobal("URL", { ...URL, createObjectURL: () => "blob:preview", revokeObjectURL: () => undefined })
		transfer.files = [file]
		expect(extractDroppedAttachments(transfer as unknown as DataTransfer, "/tmp/app").attachments).toEqual([
			{
				id: "docs/shot.png",
				path: "docs/shot.png",
				name: "shot.png",
				kind: "image",
				previewUrl: "blob:preview",
			},
		])
		vi.unstubAllGlobals()
	})

	it("没有路径的浏览器 File 不能当附件发送", () => {
		const transfer = new FakeTransfer()
		transfer.files = [new File(["x"], "shot.png", { type: "image/png" })]
		expect(extractDroppedAttachments(transfer as unknown as DataTransfer)).toEqual({
			attachments: [],
			missingPath: true,
		})
	})

	it("绝对路径列表转成工作区相对路径", () => {
		expect(attachmentsFromPaths(["/tmp/app/lib/util.go"], "/tmp/app")).toEqual([
			{ id: "lib/util.go", path: "lib/util.go", name: "util.go", kind: "file" },
		])
	})
})
