import { DownloadFile } from "@/api/modules/file"

export async function downloadAuthenticatedFile(filePath: string) {
	const data = await DownloadFile({ path: filePath })
	const downloadUrl = window.URL.createObjectURL(new Blob([data]))
	const link = document.createElement("a")
	link.href = downloadUrl
	link.download = filePath.split("/").pop() || "download"
	link.click()
	window.URL.revokeObjectURL(downloadUrl)
}
