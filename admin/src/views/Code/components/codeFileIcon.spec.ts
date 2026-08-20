import { describe, expect, it } from "vitest"
import { codeFileIcon } from "./codeFileIcon"

describe("codeFileIcon", () => {
	it("目录和常见源码用对应图标", () => {
		expect(codeFileIcon("src", true).name).toBe("mdi:folder-outline")
		expect(codeFileIcon("App.vue").name).toBe("mdi:vuejs")
		expect(codeFileIcon("admin/src/main.ts").name).toBe("mdi:language-typescript")
		expect(codeFileIcon("app/api/code.go").name).toBe("mdi:language-go")
		expect(codeFileIcon("README.md").name).toBe("mdi:language-markdown-outline")
		expect(codeFileIcon("Dockerfile").name).toBe("mdi:docker")
		expect(codeFileIcon("unknown.bin").name).toBe("mdi:file-outline")
	})
})
