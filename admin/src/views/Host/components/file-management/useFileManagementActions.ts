import type { File } from "@/api/interface/file"
import { Mimetypes } from "@/global/mimetype"
import { MsgWarning } from "@/utils/message"
import { getRandomStr } from "@/utils/util"
import { reactive, ref } from "vue"

interface UseFileManagementActionsOptions {
  t: (key: string) => string
  fileList: { value: File.File[] }
  selects: { value: string[] }
  searchParams: { value: File.ReqFile }
  uploadRef: { value: any }
  deleteRef: { value: any }
  deCompressRef: { value: any }
  compressRef: { value: any }
  createRef: { value: any }
  batchRoleRef: { value: any }
  renameRef: { value: any }
  moveRef: { value: any }
  previewRef: { value: any }
}

export const useFileManagementActions = (options: UseFileManagementActionsOptions) => {
  const {
    t,
    fileList,
    selects,
    searchParams,
    uploadRef,
    deleteRef,
    deCompressRef,
    compressRef,
    createRef,
    batchRoleRef,
    renameRef,
    moveRef,
    previewRef
  } = options

  const fileCompress = reactive({ files: [""], name: "", dst: "", operate: "compress" })
  const fileDeCompress = reactive({ files: [] as string[], path: "", name: "", dst: "", mimeType: "" })
  const filePreview = reactive({ path: "", name: "", extension: "", fileType: "" })
  const fileCreate = reactive({ path: "/", isDir: false, mode: 0o755 })
  const fileRename = reactive({ path: "", oldName: "" })
  const fileMove = reactive({ oldPaths: [""], allNames: [""], type: "", path: "", name: "", count: 0, isDir: false })
  const fileUpload = reactive({ path: "" })
  const moveOpen = ref(false)

  const pathToFiles = (paths: string[]): File.File[] => {
    const files: File.File[] = []
    for (const path of paths) {
      const file = fileList.value.find((item) => item.path === path)
      if (file) {
        files.push(file)
      }
    }
    return files
  }

  const openUpload = () => {
    fileUpload.path = searchParams.value.path
    uploadRef.value?.acceptParams(fileUpload)
  }

  const openBatchRole = (items: File.File[]) => {
    batchRoleRef.value?.acceptParams({ files: items })
  }

  const delFile = (row: File.File | null) => {
    if (deleteRef.value === null || row === null) return
    deleteRef.value.acceptParams([row])
  }

  const openDeCompress = (item: File.File) => {
    if (Mimetypes.get(item.mimeType) === undefined) {
      MsgWarning(t("file.canNotDeCompress"))
      return
    }
    fileDeCompress.name = item.name
    fileDeCompress.path = item.path
    fileDeCompress.dst = searchParams.value.path
    fileDeCompress.mimeType = item.mimeType
    deCompressRef.value?.acceptParams(fileDeCompress)
  }

  const openCompress = (items: File.File[]) => {
    const paths = items.map((item) => item.path)
    fileCompress.files = paths
    fileCompress.name = paths.length === 1 ? items[0].name : getRandomStr(6)
    fileCompress.dst = searchParams.value.path
    compressRef.value?.acceptParams(fileCompress)
  }

  const handleCreate = (command: string) => {
    fileCreate.path = searchParams.value.path
    fileCreate.isDir = command === "dir"
    createRef.value?.acceptParams(fileCreate)
  }

  const openRename = (item: File.File) => {
    fileRename.path = searchParams.value.path
    fileRename.oldName = item.name
    renameRef.value?.acceptParams(fileRename)
  }

  const batchDelFiles = () => {
    deleteRef.value?.acceptParams(pathToFiles(selects.value))
  }

  const openMove = (type: string) => {
    fileMove.type = type
    fileMove.name = ""
    fileMove.allNames = []
    fileMove.isDir = false
    fileMove.oldPaths = [...selects.value]
    fileMove.count = selects.value.length

    if (selects.value.length === 1) {
      const files = pathToFiles(selects.value)
      fileMove.name = files[0].name
      fileMove.isDir = files[0].isDir
    } else {
      fileMove.allNames = pathToFiles(selects.value).map((item) => item.name)
    }
    moveOpen.value = true
  }

  const closeMove = () => {
    selects.value = []
    fileMove.oldPaths = []
    fileMove.name = ""
    fileMove.count = 0
    fileMove.isDir = false
    moveOpen.value = false
  }

  const openPaste = () => {
    fileMove.path = searchParams.value.path
    moveRef.value?.acceptParams(fileMove)
  }

  const closeMovePage = (submit: Boolean, loadData: () => void) => {
    if (!submit) return
    loadData()
    closeMove()
  }

  const openPreview = (item: File.File, fileType: string) => {
    filePreview.path = item.isSymlink ? item.linkPath : item.path
    filePreview.name = item.name
    filePreview.extension = item.extension
    filePreview.fileType = fileType
    previewRef.value?.acceptParams(filePreview)
  }

  return {
    fileMove,
    moveOpen,
    pathToFiles,
    openUpload,
    openBatchRole,
    delFile,
    openDeCompress,
    openCompress,
    handleCreate,
    openRename,
    batchDelFiles,
    openMove,
    closeMove,
    openPaste,
    closeMovePage,
    openPreview
  }
}
