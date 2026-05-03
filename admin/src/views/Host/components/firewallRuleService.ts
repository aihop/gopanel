import {
  hostsFirewallForwardAPI,
  hostsFirewallIPAPI,
  hostsFirewallPortAPI,
  hostsFirewallUpdateAddrAPI,
  hostsFirewallUpdatePortAPI
} from "@/api/host/firewall"

type RuleType = "port" | "ip" | "forward"

export const mergeDualStackRules = (ruleType: RuleType, items: any[]) => {
  if (ruleType !== "port" && ruleType !== "ip") return items

  const map = new Map<string, any>()
  for (const item of items) {
    const port = String(item?.port ?? "")
    const protocol = String(item?.protocol ?? "")
    const strategyValue = String(item?.strategy ?? "")
    const rawAddress = String(item?.address ?? "")
    const addressNormalized = rawAddress.trim().toLowerCase() === "anywhere" ? "" : rawAddress.trim()
    const description = String(item?.description ?? "")
    const usedStatusValue = String(item?.usedStatus ?? "")
    const key = `${ruleType}|${port}|${protocol}|${strategyValue}|${addressNormalized}|${description}|${usedStatusValue}`
    const family = String(item?.family ?? "").toLowerCase()
    const existing = map.get(key)

    if (!existing) {
      const cloned = { ...item, _families: new Set<string>() }
      if (family) cloned._families.add(family)
      if (addressNormalized === "" && rawAddress.trim().toLowerCase() === "anywhere") {
        cloned.address = "Anywhere"
      }
      map.set(key, cloned)
      continue
    }

    if (family) existing._families.add(family)
    if (!existing.address && rawAddress.trim().toLowerCase() === "anywhere") {
      existing.address = "Anywhere"
    }
  }

  return Array.from(map.values()).map((item) => {
    const families: Set<string> | undefined = item._families
    if (families && families.size > 0) {
      const list = Array.from(families)
      const hasV4 = families.has("ipv4")
      const hasV6 = families.has("ipv6")
      item.family = hasV4 && hasV6 ? "ipv4/ipv6" : list[0]
    }
    delete item._families
    return item
  })
}

export const deleteFirewallRule = (ruleType: RuleType, row: any) => {
  if (ruleType === "ip") {
    return hostsFirewallIPAPI({
      operation: "remove",
      address: row.address,
      strategy: row.strategy,
      description: row.description || ""
    })
  }

  if (ruleType === "forward") {
    return hostsFirewallForwardAPI({
      forceDelete: false,
      rules: [
        {
          operation: "remove",
          num: row.num || "",
          protocol: row.protocol,
          port: row.port,
          targetIP: row.targetIP,
          targetPort: row.targetPort
        }
      ]
    })
  }

  return hostsFirewallPortAPI({
    operation: "remove",
    port: row.port,
    protocol: row.protocol,
    strategy: row.strategy,
    address: row.address || "",
    description: row.description || ""
  })
}

export const saveFirewallRule = async (payload: any) => {
  if (payload.type === "ip") {
    if (payload.isEdit) {
      return hostsFirewallUpdateAddrAPI({
        oldRule: {
          operation: "remove",
          address: payload.oldData?.address || "",
          strategy: payload.oldData?.strategy || "accept",
          description: payload.oldData?.description || ""
        },
        newRule: {
          operation: "add",
          address: payload.data.address,
          strategy: payload.data.strategy,
          description: payload.data.description || ""
        }
      })
    }

    return hostsFirewallIPAPI({
      operation: "add",
      address: payload.data.address,
      strategy: payload.data.strategy,
      description: payload.data.description || ""
    })
  }

  if (payload.type === "forward") {
    let res = await hostsFirewallForwardAPI({
      forceDelete: false,
      rules: [
        {
          operation: payload.isEdit ? "remove" : "add",
          num: payload.isEdit ? payload.oldData?.num || "" : "",
          protocol: payload.isEdit ? payload.oldData?.protocol || "tcp" : payload.data.protocol,
          port: payload.isEdit ? payload.oldData?.port || "" : payload.data.port,
          targetIP: payload.isEdit ? payload.oldData?.targetIP || "127.0.0.1" : payload.data.targetIP,
          targetPort: payload.isEdit ? payload.oldData?.targetPort || "" : payload.data.targetPort
        }
      ]
    })

    if (payload.isEdit && res?.code === 0) {
      res = await hostsFirewallForwardAPI({
        forceDelete: false,
        rules: [
          {
            operation: "add",
            protocol: payload.data.protocol,
            port: payload.data.port,
            targetIP: payload.data.targetIP,
            targetPort: payload.data.targetPort
          }
        ]
      })
    }

    return res
  }

  if (payload.isEdit) {
    return hostsFirewallUpdatePortAPI({
      oldRule: {
        operation: "remove",
        port: payload.oldData?.port || "",
        protocol: payload.oldData?.protocol || "tcp",
        strategy: payload.oldData?.strategy || "accept",
        address: payload.oldData?.address || "",
        description: payload.oldData?.description || ""
      },
      newRule: {
        operation: "add",
        port: payload.data.port,
        protocol: payload.data.protocol,
        strategy: payload.data.strategy,
        address: payload.data.address || "",
        description: payload.data.description || ""
      }
    })
  }

  return hostsFirewallPortAPI({
    operation: "add",
    port: payload.data.port,
    protocol: payload.data.protocol,
    strategy: payload.data.strategy,
    address: payload.data.address || "",
    description: payload.data.description || ""
  })
}
