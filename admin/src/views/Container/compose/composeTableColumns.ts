import { h } from "vue"
import { NButton, NSpace, type DataTableColumns } from "naive-ui"
import { t } from "@/i18n"
import type { RowData } from "./composeTypes"

export const createComposeColumns = (options: {
  edit: (row: RowData) => void
  remove: (row: RowData) => void
}): DataTableColumns<RowData> => [
  {
    title: t("commons.table.name"),
    key: "name",
    render(row) {
      return h("span", row.name)
    }
  },
  {
    title: t("database.source"),
    key: "source"
  },
  {
    title: t("container.composeDirectory"),
    key: "directory",
    align: "center"
  },
  {
    title: t("container.containerStatus"),
    key: "status"
  },
  {
    title: t("commons.table.createdAt"),
    key: "createdTime"
  },
  {
    title: t("database.actions"),
    key: "actions",
    render(row) {
      return h(NSpace, {}, {
        default: () => [
          h(
            NButton,
            {
              text: true,
              tertiary: true,
              type: "primary",
              onClick: () => options.edit(row)
            },
            { default: () => "编辑" }
          ),
          h(
            NButton,
            {
              text: true,
              tertiary: true,
              type: "error",
              onClick: () => options.remove(row)
            },
            { default: () => "删除" }
          )
        ]
      })
    }
  }
]
