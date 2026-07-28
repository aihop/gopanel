import type { File } from "@/api/interface/file"
import type { DataTableColumns } from "naive-ui"
import { ComputeDirSize } from "@/api/modules/file"
import { Mimetypes } from "@/global/mimetype"
import { formatTime } from "@/utils/date"
import { computeSize, copyText } from "@/utils/util"
import { downloadAuthenticatedFile } from "@/utils/fileDownload"
import { h, reactive } from "vue"
import { NButton, NDropdown, NSpace } from "naive-ui"

type Translate = (key: string) => string

interface FileTableColumnOptions {
  t: Translate
  getAuth: () => string | undefined
  onEnterDirectory: (row: File.File) => void
  onOpenView: (row: File.File) => void
  onDelete: (row: File.File) => void
  onBatchRole: (items: File.File[]) => void
  onCompress: (items: File.File[]) => void
  onDecompress: (row: File.File) => void
  onRename: (row: File.File) => void
  onError: (message: string) => void
}

export const createFileTableColumns = (options: FileTableColumnOptions): DataTableColumns<File.File> => {
  const {
    t,
    getAuth,
    onEnterDirectory,
    onOpenView,
    onDelete,
    onBatchRole,
    onCompress,
    onDecompress,
    onRename,
    onError
  } = options

  // 已计算的目录大小缓存 { [path]: size }
  const dirSizeMap = reactive({} as Record<string, number>)
  // 正在计算中的目录 { [path]: true }
  const dirSizeLoadingMap = reactive({} as Record<string, boolean>)

  const handleDownload = async (row: File.File) => {
    if (row.isDir || !row.path) return
    if (!getAuth()) {
      onError(t("file.downloadError"))
      return
    }
    try {
      await downloadAuthenticatedFile(row.path)
    } catch {
      onError(t("file.downloadError"))
    }
  }

  const handleDirSize = async (row: File.File) => {
    if (!row.isDir || !row.path) return
    if (dirSizeLoadingMap[row.path]) return // 已经在计算中
    dirSizeLoadingMap[row.path] = true
    try {
      const res = await ComputeDirSize({ path: row.path })
      const size = res?.data?.size
      if (size !== undefined && size !== null) {
        dirSizeMap[row.path] = size
      }
    } catch {
      // 静默失败，保留按钮状态
    } finally {
      dirSizeLoadingMap[row.path] = false
    }
  }

  return [
    {
      type: "selection" as const
    },
    {
      title: "名称",
      key: "name",
      render(row: File.File) {
        return h(
          NSpace,
          { align: "center" },
          {
            default: () => [
              h("span", { style: { marginRight: "8px", fontSize: "16px" } }, row.isDir ? "📁" : "📄"),
              h(
                "span",
                {
                  style: {
                    cursor: "pointer",
                    color: "#005eeb"
                  },
                  onClick: row.isDir ? () => onEnterDirectory(row) : () => onOpenView(row)
                },
                row.name
              )
            ]
          }
        )
      }
    },
    {
      title: "大小",
      key: "size",
      render(row: File.File) {
        if (!row.isDir) {
          return h("span", computeSize(row.size))
        }
        const cached = dirSizeMap[row.path]
        if (cached !== undefined) {
          return h("span", computeSize(cached))
        }
        const loading = dirSizeLoadingMap[row.path]
        return h(
          NButton,
          {
            size: "tiny",
            text: true,
            type: "primary",
            loading: loading,
            disabled: loading,
            onClick: () => handleDirSize(row)
          },
          { default: () => t("file.calcDirSize") }
        )
      }
    },
    {
      title: "权限",
      key: "mode",
      render(row: File.File) {
        return h("span", row.mode)
      }
    },
    {
      title: "所有者",
      key: "user",
      render(row: File.File) {
        return h("span", `${row.user}:${row.group}`)
      }
    },
    {
      title: "修改时间",
      key: "modTime",
      render(row: File.File) {
        return h("span", formatTime(row.modTime))
      }
    },
    {
      title: "操作",
      key: "actions",
      fixed: "right" as const,
      render(row: File.File) {
        return h(
          NSpace,
          { size: "small" },
          {
            default: () => [
              h(
                NButton,
                {
                  size: "small",
                  type: "primary",
                  text: true,
                  onClick: () => {
                    if (row.isDir) {
                      onEnterDirectory(row)
                      return
                    }
                    onOpenView(row)
                  }
                },
                { default: () => t("file.open") }
              ),
              h(
                NButton,
                {
                  size: "small",
                  type: "primary",
                  disabled: !!row.isDir || !row.path,
                  text: true,
                  onClick: () => handleDownload(row)
                },
                { default: () => t("file.download") }
              ),
              h(
                NDropdown,
                {
                  options: [
                    { label: t("file.copyDir"), key: "copyDir" },
                    { label: t("file.editPermissions"), key: "batchRole" },
                    { label: t("file.rename"), key: "rename" },
                    { label: t("file.compress"), key: "compress" },
                    {
                      label: t("file.deCompress"),
                      key: "decompress",
                      disabled: Mimetypes.get(row.mimeType) === undefined || row.isDir
                    },
                    { label: t("commons.button.delete"), key: "delete" }
                  ],
                  onSelect: async (key) => {
                    switch (key) {
                      case "copyDir":
                        if (row?.path) {
                          await copyText(row.path)
                        }
                        break
                      case "delete":
                        onDelete(row)
                        break
                      case "batchRole":
                        onBatchRole([row])
                        break
                      case "compress":
                        onCompress([row])
                        break
                      case "decompress":
                        onDecompress(row)
                        break
                      case "rename":
                        onRename(row)
                        break
                    }
                  }
                },
                {
                  default: () =>
                    h(
                      NButton,
                      {
                        size: "small",
                        type: "primary",
                        text: true
                      },
                      { default: () => t("tabs.more") }
                    )
                }
              )
            ]
          }
        )
      }
    }
  ]
}
