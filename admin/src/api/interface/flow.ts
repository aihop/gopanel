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
		pipelineSourceType: "git" | "code"
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

	export interface UpdateInput {
		name: string
		pipelineId: number
		autoStartAfterCodeDelivery: boolean
		environments: CreateEnvironment[]
	}

	export type RunStatus = "queued" | "running" | "success" | "failed" | "waiting_deployment"
	export type RunStage = "created" | "building" | "publishing" | "release_ready" | "deploying" | "deployed" | "waiting_deployment" | "failed"

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
		sourceType: "git" | "code_delivery" | "code_baseline"
		sourceCommit: string
		sourceDigest: string
		sourceTaskTitle: string
		sourceRepositories?: SourceRepository[]
		codeDeliveryJobId: number
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
		codeDeliveryJobId?: number
		useProjectBaseline?: boolean
		sourceCommit?: string
		version?: string
	}

	export interface SourceRepository {
		name: string
		workspacePath: string
		targetBranch: string
		commit: string
	}

	export interface CodeDeliverySource {
		jobId: number
		sessionId: number
		taskId: number
		taskTitle: string
		completedAt?: string
		sourceDigest: string
		repositories: SourceRepository[]
	}

	export interface CodeBaselineSource {
		available: boolean
		sourceDigest?: string
		hasUncommittedChanges: boolean
		repositories: SourceRepository[]
	}
}
