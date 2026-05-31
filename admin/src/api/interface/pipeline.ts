export namespace Pipeline {
  export interface ReqCreate {
    name: string;
    description?: string;
    repoUrl?: string; // 选填
    branch: string;
    version: string; // 新增版本号
    authType: string;
    authData?: string;
    buildImage: string;
    buildScript: string;
    outputImage?: string;
    artifactPath: string;
    exposePort?: number;
    pipelineKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
    actionType?: string;
    actionParams?: string;
    autoDeploy?: boolean;
  }

  export interface ReqUpdate extends ReqCreate {
    id: number;
  }

  export interface ResPipeline {
    id: number;
    createdAt: string;
    updatedAt: string;
    name: string;
    description: string;
    repoUrl: string;
    branch: string;
    version: string; // 新增版本号
    authType: string;
    authData: string;
    buildImage: string;
    buildScript: string;
    outputImage?: string;
    artifactPath: string;
    exposePort: number;
    pipelineKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
    actionType?: string;
    actionParams?: string;
    autoDeploy?: boolean;
    runtimeHost?: string;
    runtimeKind?: string;
    runtimeMode?: string;
    runUser?: string;
  }

  export interface ResRecord {
    id: number;
    createdAt: string;
    updatedAt: string;
    pipelineId: number;
    status: string;
    version: string; // 新增版本号
    commitHash?: string;
    errorMessage: string;
    archiveFile: string;
    imageTag?: string;
    runnerReleaseDir?: string;
    runnerContainerId?: string;
    runnerHostPort?: number;
    runtimeHost?: string;
    runtimeKind?: string;
    runtimeMode?: string;
    runUser?: string;
    released?: boolean;
    activeWebsiteCount?: number;
    activeWebsiteNames?: string[];
  }

  export interface ResRelease {
    id: number;
    createdAt: string;
    updatedAt: string;
    pipelineId: number;
    pipelineRecordId: number;
    version: string;
    commitHash?: string;
    sourceType: string;
    imageTag?: string;
    archiveFile?: string;
    releaseDir?: string;
    artifactMeta?: string;
    status: string;
    remark?: string;
    activeWebsiteCount?: number;
    activeWebsiteNames?: string[];
  }

  export interface ReqDetectRunnerPreset {
    repoUrl: string;
    branch: string;
    authType?: string;
    authData?: string;
  }

  export interface ResDetectRunnerPreset {
    preset: string;
    hits: string[];
  }
}
