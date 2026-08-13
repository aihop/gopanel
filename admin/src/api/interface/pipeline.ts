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
    artifactPath: string;
    pipelineKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
    actionType?: string;
    actionParams?: Record<string, any>;
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
    artifactPath: string;
    pipelineKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
    actionType?: string;
    actionParams?: Record<string, any>;
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
    expectedCommit?: string;
    commitHash?: string;
    changelog?: string; // 本次构建包含的提交标题，一行一条
    errorMessage: string;
    archiveFile: string;
    imageTag?: string;
    imageId?: string;
    imageDigest?: string;
    imageRef?: string;
    runnerReleaseDir?: string;
    runnerContainerId?: string;
    runnerHostPort?: number;
    runtimeHost?: string;
    runtimeKind?: string;
    runtimeMode?: string;
    runUser?: string;
    released?: boolean;
  }

  export interface ResRelease {
    id: number;
    createdAt: string;
    updatedAt: string;
    pipelineId: number;
    pipelineRecordId: number;
    version: string;
    commitHash?: string;
    changelog?: string; // 发布时从构建记录复制的更新说明
    sourceType: string;
    imageTag?: string;
    imageDigest?: string;
    archiveFile?: string;
    releaseDir?: string;
    artifactDigest?: string;
    artifactManifest?: string;
    artifactMeta?: string;
    status: string;
    remark?: string;
  }

  export interface ResForceDelete {
    pipelineId: number;
    recordCount: number;
    releaseCount: number;
    cleanupWarnings: string[];
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
