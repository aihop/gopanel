import { h } from "vue"
import { NButton, NInput, NSelect } from "naive-ui"

export const createExposedPortColumns = (
  t: (key: string) => string,
  getPorts: () => Array<any>,
  onDelete: (index: number) => void
) => [
  {
    title: () => t("container.server"),
    key: "host",
    minWidth: 150,
    render(row: any, index: number) {
      return h(NInput, {
        placeholder: t("container.serverExample"),
        value: row.host,
        "onUpdate:value": v => {
          getPorts()[index].host = v
        }
      })
    }
  },
  {
    title: () => t("container.container"),
    key: "containerPort",
    minWidth: 80,
    render(row: any, index: number) {
      return h(NInput, {
        placeholder: t("container.containerExample"),
        value: row.containerPort,
        "onUpdate:value": v => {
          getPorts()[index].containerPort = v
        }
      })
    }
  },
  {
    title: () => t("commons.table.protocol"),
    key: "protocol",
    minWidth: 50,
    render(row: any, index: number) {
      return h(NSelect, {
        value: row.protocol,
        options: [
          { label: "tcp", value: "tcp" },
          { label: "udp", value: "udp" }
        ],
        class: "w-full",
        placeholder: t("container.serverExample"),
        "onUpdate:value": v => {
          getPorts()[index].protocol = v
        }
      })
    }
  },
  {
    title: "",
    key: "actions",
    minWidth: 35,
    render(_row: any, index: number) {
      return h(
        NButton,
        {
          text: true,
          type: "primary",
          onClick: () => onDelete(index)
        },
        { default: () => t("commons.button.delete") }
      )
    }
  }
]
