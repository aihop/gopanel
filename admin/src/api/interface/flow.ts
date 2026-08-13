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
}
