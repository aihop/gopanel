import type { File } from "@/api/interface/file"
import type { DataTableColumns } from "naive-ui"
import { userTokenAPI } from "@/api/modules/user"
import { Mimetypes } from "@/global/mimetype"
import { formatTime } from "@/utils/date"
import { computeSize, copyText, downloadFile } from "@/utils/util"
import { h } from "vue"
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
  onDirSize: (row: File.File) => void
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
    onDirSize,
    onError
  } = options

  const copyAuthorizedDownloadLink = async (row: File.File) => {
    if (!row?.path) return
    try {
      const res = await userTokenAPI({ path: row.path, timestamp: Date.now() })
      if (res.code !== 0) {
        onError(res.msg || t("file.downloadError"))
        return
      }
      const token = res.data
      if (!token) {
        onError(t("file.downloadError"))
        return
      }
      const href = window.location.href
      const protocol = href.split("//")[0]
      const host = href.split("//")[1].split("/")[0]
      const url = `${protocol}//${host}/api/file/download?token=${token}&path=${encodeURIComponent(row.path)}`
      await copyText(url)
    } catch (error) {
      onError(t("file.downloadError"))
    }
  }

  const handleDownload = (row: File.File) => {
    if (row.isDir || !row.path) return
    const auth = getAuth()
    if (!auth) {
      onError(t("file.downloadError"))
      return
    }
    downloadFile(row.path, auth)
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
        return h("span", row.isDir ? "-" : computeSize(row.size))
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
                    { label: t("file.calcDirSize"), key: "dirSize", disabled: !row.isDir },
                    { label: t("file.rename"), key: "rename" },
                    { label: t("file.compress"), key: "compress" },
                    {
                      label: t("file.deCompress"),
                      key: "decompress",
                      disabled: Mimetypes.get(row.mimeType) === undefined || row.isDir
                    },
                    { label: t("commons.button.delete"), key: "delete" },
                    { label: "授权下载链接", key: "copyAuthDownload" }
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
                      case "dirSize":
                        onDirSize(row)
                        break
                      case "rename":
                        onRename(row)
                        break
                      case "copyAuthDownload":
                        await copyAuthorizedDownloadLink(row)
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
