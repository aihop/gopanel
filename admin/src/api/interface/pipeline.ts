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
    runnerKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
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
    runnerKey?: string;
    runnerMode?: string;
    runnerConfig?: any;
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
    runnerReleaseDir?: string;
    runnerContainerId?: string;
    runnerHostPort?: number;
  }
}
