import type { DataTableColumns } from "naive-ui"
import type { Website } from "@/api/interface/website"
import { h } from "vue"
import { NButton, NDropdown, NTag, NTooltip } from "naive-ui"
import { formatDateTime, isExpired, sourceLabel, type SSLRow } from "./sslHelpers"

interface SSLColumnOptions {
  buildBoundWebsiteRuntimeText: (item: { id: number; name?: string; primaryDomain?: string }) => string
  openDetail: (id: number) => void
  openPushCDNModal: (row: SSLRow) => void
  openPushRuleModal: (row: SSLRow) => void
  openLogModal: (id: number) => void
  handleRenewCertificate: (row: SSLRow) => void
  downloadContent: (content: string, fileName: string) => void
  openApplyModal: (sslId: number) => void
  confirmDelete: (row: SSLRow) => void
}

interface PushRuleColumnOptions {
  getCloudAccountLabel: (cloudAccountId: number) => string
  onEdit: (row: Website.SSLPushRule) => void
  onDelete: (row: Website.SSLPushRule) => void
}

export const createSSLColumns = (options: SSLColumnOptions): DataTableColumns<SSLRow> => [
  {
    title: "主域名",
    key: "primaryDomain",
    width: 150
  },
  {
    title: "域名列表",
    key: "domains",
    render(row) {
      const domains = (row.domains || "")
        .split(",")
        .map(item => item.trim())
        .filter(Boolean)
      return h(
        "div",
        { class: "flex flex-wrap gap-2" },
        domains.length
          ? domains.map(domain =>
              h("span", { class: "rounded-full bg-slate-100 px-3 py-1 text-xs text-slate-600" }, domain)
            )
          : [h("span", { class: "text-slate-400" }, "--")]
      )
    }
  },
  {
    title: "来源",
    key: "type",
    width: 160,
    render(row) {
      const info = sourceLabel(row)
      return h(NTag, { type: info.tagType, bordered: false, round: true, size: "small" }, { default: () => info.label })
    }
  },
  {
    title: "绑定网站",
    key: "websites",
    render(row) {
      const items = row.websites || []
      if (!items.length) {
        return h("span", { class: "text-slate-400" }, "未绑定")
      }
      return h(
        "div",
        { class: "flex flex-wrap gap-2" },
        items.map(item =>
          h(
            NTooltip,
            null,
            {
              trigger: () =>
                h(
                  "span",
                  { class: "rounded-full bg-blue-50 px-3 py-1 text-xs text-blue-600" },
                  item.name || item.primaryDomain || `#${item.id}`
                ),
              default: () => options.buildBoundWebsiteRuntimeText(item)
            }
          )
        )
      )
    }
  },
  {
    title: "颁发者",
    key: "organization",
    width: 130,
    render(row) {
      return row.organization || "--"
    }
  },
  {
    title: "到期时间",
    key: "expireDate",
    width: 180,
    render(row) {
      return formatDateTime(row.expireDate)
    }
  },
  {
    title: "状态",
    key: "status",
    width: 80,
    fixed: "right",
    render(row) {
      const expired = isExpired(row.expireDate)
      return h(NTag, { type: expired ? "error" : "success", bordered: false, round: true }, {
        default: () => (expired ? "已过期" : "有效")
      })
    }
  },
  {
    title: "操作",
    key: "actions",
    width: 220,
    fixed: "right",
    render(row) {
      const moreOptions = [
        { label: "下载证书", key: "download-crt" },
        { label: "下载私钥", key: "download-key" },
        { label: "删除", key: "delete" }
      ]

      if (row.type === "upload") {
        moreOptions.splice(2, 0, { label: "绑定网站", key: "apply" })
      }
      if (isExpired(row.expireDate) && row.type !== "upload") {
        moreOptions.unshift({ label: "立即签注", key: "renew" })
      }
      if (row.status === "pending") {
        moreOptions.unshift({ label: "查看日志", key: "view-log" })
      }

      return h("div", { class: "flex items-center gap-2" }, [
        h(
          NButton,
          { size: "small", text: true, type: "primary", onClick: () => options.openDetail(row.id) },
          { default: () => "详情" }
        ),
        h(
          NButton,
          { size: "small", text: true, type: "primary", onClick: () => options.openPushCDNModal(row) },
          { default: () => "推送CDN" }
        ),
        h(
          NButton,
          { size: "small", text: true, type: "primary", onClick: () => options.openPushRuleModal(row) },
          { default: () => "自动部署" }
        ),
        h(
          NDropdown,
          {
            options: moreOptions,
            onSelect: (key: string) => {
              switch (key) {
                case "view-log":
                  options.openLogModal(row.id)
                  break
                case "renew":
                  options.handleRenewCertificate(row)
                  break
                case "download-crt":
                  options.downloadContent(row.pem, `${row.primaryDomain}.crt`)
                  break
                case "download-key":
                  options.downloadContent(row.privateKey, `${row.primaryDomain}.key`)
                  break
                case "apply":
                  options.openApplyModal(row.id)
                  break
                case "delete":
                  options.confirmDelete(row)
                  break
              }
            }
          },
          {
            default: () => h(NButton, { size: "small", text: true, type: "primary" }, { default: () => "更多" })
          }
        )
      ])
    }
  }
]

export const createPushRuleColumns = (options: PushRuleColumnOptions): DataTableColumns<Website.SSLPushRule> => [
  {
    title: "云账号",
    key: "cloudAccountId",
    render(row) {
      return options.getCloudAccountLabel(row.cloudAccountId)
    }
  },
  {
    title: "目标域名",
    key: "targetDomain",
    render(row) {
      return row.targetDomain || "（默认主域名）"
    }
  },
  {
    title: "操作",
    key: "actions",
    width: 140,
    render(row) {
      return h("div", { class: "flex gap-2" }, [
        h(NButton, { size: "small", text: true, type: "primary", onClick: () => options.onEdit(row) }, { default: () => "编辑" }),
        h(NButton, { size: "small", text: true, type: "error", onClick: () => options.onDelete(row) }, { default: () => "删除" })
      ])
    }
  }
]
