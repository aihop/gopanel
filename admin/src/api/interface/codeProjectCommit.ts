export interface CodeProjectCommitResult {
	repositoryName: string
	/** committed = 已提交；clean = 本来就干净或改动全被忽略；failed = 失败 */
	status: "committed" | "clean" | "failed"
	commit?: string
	files: number
	errorMessage?: string
}
