import { getPipelinePage } from "@/api/modules/pipeline"
import type { Pipeline } from "@/api/interface/pipeline"

export async function listAllPipelines(limit = 200): Promise<Pipeline.ResPipeline[]> {
	const items: Pipeline.ResPipeline[] = []
	let page = 1
	let total = 0

	do {
		const res: any = await getPipelinePage({ page, limit })
		const pageItems = Array.isArray(res?.data?.items) ? res.data.items : []
		total = Number(res?.data?.total || 0)
		items.push(...pageItems)
		if (pageItems.length === 0) break
		page += 1
	} while (items.length < total)

	return items
}
