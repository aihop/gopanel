import type { File } from "@/api/interface/file"
import { computed } from "vue"

interface UseFilePathNavigationOptions {
  searchParams: { value: File.ReqFile }
  loadData: () => void
}

export const useFilePathNavigation = ({ searchParams, loadData }: UseFilePathNavigationOptions) => {
  const handleSearch = () => {
    searchParams.value.page = 1
    loadData()
  }

  const pathSegments = computed(() => {
    const segments = searchParams.value.path.split("/").filter((segment, index) => index === 0 || segment)
    if (segments[0] !== "") segments.unshift("")
    return segments
  })

  const goToParentDirectory = () => {
    if (searchParams.value.path === "/") return
    const parentPath = searchParams.value.path.substring(0, searchParams.value.path.lastIndexOf("/"))
    searchParams.value.path = parentPath || "/"
    searchParams.value.page = 1
    loadData()
  }

  const goToPath = (index: number) => {
    let newPath = "/"
    if (index >= 0) {
      newPath += pathSegments.value.slice(1, index + 1).join("/")
    }
    searchParams.value.path = newPath
    searchParams.value.page = 1
    searchParams.value.search = ""
    loadData()
  }

  return {
    handleSearch,
    pathSegments,
    goToParentDirectory,
    goToPath
  }
}
