export namespace Flow {
	export interface Environment {
		id: number
		flowId: number
		name: "preview" | "production"
		websiteId: number
		websiteName: string
		autoDeploy: boolean
		approvalRequired: boolean
		enabled: boolean
	}

	export interface Item {
		id: number
		createdAt: string
		updatedAt: string
		projectId: number
		projectName: string
		name: string
		pipelineId: number
		pipelineName: string
		enabled: boolean
		autoStartAfterCodeDelivery: boolean
		createdBy: number
		environments: Environment[]
	}

	export interface CreateEnvironment {
		name: "preview" | "production"
		websiteId: number
		autoDeploy: boolean
		approvalRequired: boolean
	}

	export interface CreateInput {
		name: string
		projectId: number
		pipelineId: number
		autoStartAfterCodeDelivery: boolean
		environments: CreateEnvironment[]
	}

	export type RunStatus = "queued" | "running" | "failed" | "waiting_deployment"
	export type RunStage = "created" | "building" | "publishing" | "release_ready" | "waiting_deployment" | "failed"

	export interface StageRun {
		id: number
		flowRunId: number
		stage: RunStage
		attempt: number
		status: "pending" | "running" | "success" | "failed"
		resourceType: string
		resourceId: number
		summary: string
		errorCode: string
		errorDetail: string
		startedAt?: string
		completedAt?: string
	}

	export interface Run {
		id: number
		createdAt: string
		updatedAt: string
		flowId: number
		flowName: string
		projectId: number
		projectName: string
		pipelineId: number
		pipelineName: string
		version: string
		sourceBranch: string
		sourceCommit: string
		pipelineRecordId: number
		releaseId: number
		artifactDigest: string
		currentStage: RunStage
		status: RunStatus
		failureCode: string
		errorSummary: string
		startedAt?: string
		completedAt?: string
		stages?: StageRun[]
	}

	export interface RunCreateInput {
		flowId: number
		sourceCommit: string
		version?: string
	}
}
