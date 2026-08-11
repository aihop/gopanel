import type { CodeGitDiffKind, CodeGitFile, CodeGitRepository, CodeGitStatus } from "@/api/interface/codeGit"

export interface CodeGitReviewEntry {
	repository: CodeGitRepository
	file: CodeGitFile
	kind: CodeGitDiffKind
	key: string
}

export function codeGitReviewEntries(status: CodeGitStatus | null) {
	const entries: CodeGitReviewEntry[] = []
	for (const repository of status?.repositories || []) {
		for (const file of repository.files) {
			if (status?.scope === "result") {
				entries.push({ repository, file, kind: "result", key: `${repository.id}:result:${file.path}` })
				continue
			}
			if (file.staged) {
				entries.push({ repository, file, kind: "staged", key: `${repository.id}:staged:${file.path}` })
			}
			if (file.changed || file.untracked) {
				entries.push({ repository, file, kind: "working", key: `${repository.id}:working:${file.path}` })
			}
		}
	}
	return entries
}

export function codeGitReviewTotals(status: CodeGitStatus | null) {
	return {
		additions: (status?.additions || 0) + (status?.stagedAdditions || 0),
		deletions: (status?.deletions || 0) + (status?.stagedDeletions || 0)
	}
}
