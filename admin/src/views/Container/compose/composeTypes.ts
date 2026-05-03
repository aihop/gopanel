export type RowData = {
  key: number
  name: string
  source: string
  directory: string
  status: string
  createdTime: string
  containerNumber: number
  configFile: string
  workdir: string
  path: string
  containers: Array<{
    containerID: string
    name: string
    createTime: string
    state: string
  }>
  env: string | null
}

export const initialComposeFormState = {
  source: "editor",
  projectName: "",
  composeContent: "",
  envContent: "",
  pathValue: "",
  selectedTemplateId: null as string | null
}
