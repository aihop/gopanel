import { useRouter } from "vue-router"

const router = useRouter()

export function toFolder(folder: string) {
	router.push({ path: "/hosts/files", query: { path: folder } })
}
