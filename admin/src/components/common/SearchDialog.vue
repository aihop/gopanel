<template>
  <!-- eslint-disable vue/no-v-model-argument -->
  <n-modal
    v-model:show="showSearchBox"
    class="search-box-modal"
  >
    <n-card
      content-class="!p-0"
      class="!w-150"
      :bordered="false"
      size="huge"
      role="dialog"
      aria-modal="true"
    >
      <div
        class="search-box"
        @keydown.up="prevItem()"
        @keydown.down="nextItem()"
      >
        <div class="search-input flex items-center">
          <Icon
            :name="SearchIcon"
            :size="16"
          />
          <input
            v-model="search"
            placeholder="Search"
            class="grow"
          />
          <n-text code>ESC</n-text>
          <Icon
            :name="CloseIcon"
            :size="20"
            class="cursor-pointer"
            @click="closeBox()"
          />
        </div>
        <n-divider />
        <n-scrollbar
          ref="scrollContent"
          class="!h-96"
        >
          <div class="conten-wrap">
            <div
              v-for="group of filteredGroups"
              :key="group.name"
              class="group"
            >
              <div class="group-title">{{ group.name }}</div>
              <div class="group-list">
                <button
                  v-for="item of group.items"
                  :id="item.key.toString()"
                  :key="item.key"
                  class="item flex items-center"
                  :class="{ active: item.key === activeItem }"
                  @click="callAction(item.action)"
                >
                  <div class="icon">
                    <n-avatar
                      v-if="item.iconImage"
                      round
                      :size="28"
                      :src="item.iconImage"
                      :img-props="{ alt: 'avatar' }"
                    />
                    <Icon
                      v-if="item.iconName"
                      :name="item.iconName"
                      :size="16"
                    />
                  </div>
                  <div class="title grow">
                    <div v-html="getHighlighterText(item.title)"></div>
                  </div>
                  <div class="label">{{ item.label }}</div>
                </button>
              </div>
            </div>
            <div
              v-if="!filteredGroups.length"
              class="group-empty"
            >
              We couldn't find anything matching "{{ search }}"
            </div>
          </div>
        </n-scrollbar>
        <n-divider />
        <div class="hint-bar flex items-center justify-center">
          <div class="hint flex items-center justify-center gap-1">
            <div class="icon">
              <Icon
                :name="ArrowEnterIcon"
                :size="12"
              />
            </div>
            <span class="label">to select</span>
          </div>
          <div class="hint flex items-center justify-center gap-1">
            <div class="icon">
              <Icon
                :name="ArrowSortIcon"
                :size="12"
              />
            </div>
            <span class="label">to navigate</span>
          </div>
        </div>
      </div>
    </n-card>
  </n-modal>
</template>

<script lang="ts" setup>
// @ts-nocheck
import type { ScrollbarInst } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { useFullscreenSwitch } from "@/composables/useFullscreenSwitch"
import { useSearchDialog } from "@/composables/useSearchDialog"
import { useThemeSwitch } from "@/composables/useThemeSwitch"
import { getOS } from "@/utils"
import { useMagicKeys, whenever } from "@vueuse/core"
import { NAvatar, NCard, NDivider, NModal, NScrollbar, NText } from "naive-ui"
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"
import { useAuthStore } from "@/store/auth"
import { t } from "@/i18n"

interface GroupItem {
	iconName: string | null
	iconImage: string | null
	key: string
	title: string
	label: string
	tags?: string[]
	action: () => void
}

interface Group {
	name: string
	items: GroupItem[]
}

type Groups = Group[]

const SearchIcon = "ion:search-outline"
const TodoIcon = "fluent:task-list-square-add-20-regular"
const EmailIcon = "fluent:mail-edit-20-regular"
const NotesIcon = "fluent:chart-person-20-regular"
const ArrowEnterIcon = "fluent:arrow-enter-left-24-regular"
const ArrowSortIcon = "fluent:arrow-sort-24-regular"
const FullScreenIcon = "fluent:full-screen-maximize-24-regular"
const DarkModeIcon = "ion:moon-outline"
const CloseIcon = "ion:close"

const router = useRouter()
const authStore = useAuthStore()
const showSearchBox = ref(false)
const search = ref("")
const activeItem = ref<null | string>(null)
const commandIcon = ref("⌘")
const scrollContent = ref<(ScrollbarInst & { $el: HTMLElement }) | null>(null)

const groups = ref<Groups>([
	{
		name: "快速导航",
		items: [
			{
				iconName: "mdi:view-dashboard-outline",
				iconImage: null,
				key: "dashboard",
				title: "概览看板",
				label: "快捷",
				tags: ["dashboard", "home", "index", "概览", "首页", "看板"],
				action() {
					router.push({ name: "Dashboard-Index" })
				}
			},
			{
				iconName: "mdi:robot-outline",
				iconImage: null,
				key: "ai",
				title: "AI 助手",
				label: "快捷",
				tags: ["ai", "agent", "terminal", "终端", "智能体"],
				action() {
					router.push({ name: "AIAgent-Index" })
				}
			},
			{
				iconName: "ion:settings-outline",
				iconImage: null,
				key: "setting",
				title: "系统设置",
				label: "快捷",
				tags: ["setting", "config", "设置", "配置", "系统"],
				action() {
					router.push({ name: "Setting" })
				}
			}
		]
	},
	{
		name: "网站与应用",
		items: [
			{
				iconName: "mdi:web",
				iconImage: null,
				key: "Website-Index",
				title: t("menu.website"),
				label: t("menu.website"),
				tags: ["website", "list", "网站", "站点"],
				action() {
					router.push({ name: "Website-Index" })
				}
			},
			{
				iconName: "mdi:shield-check-outline",
				iconImage: null,
				key: "ssl",
				title: "SSL 证书",
				label: "证书",
				tags: ["ssl", "https", "证书", "安全"],
				action() {
					router.push({ name: "SSL-Index" })
				}
			},
			{
				iconName: "mdi:apps",
				iconImage: null,
				key: "apps",
				title: "应用",
				label: "应用",
				tags: ["apps", "store", "应用", "商店", "插件"],
				action() {
					router.push({ name: "Apps-Index" })
				}
			}
		]
	},
	{
		name: "服务与资源",
		items: [
			{
				iconName: "mdi:database",
				iconImage: null,
				key: "database",
				title: "数据库",
				label: "服务",
				tags: ["database", "mysql", "redis", "postgresql", "数据库"],
				action() {
					router.push({ name: "Database-Index" })
				}
			},
			{
				iconName: "mdi:docker",
				iconImage: null,
				key: "container",
				title: "Docker 容器",
				label: t("menu.container"),
				tags: ["docker", "container", "compose", t("menu.container")],
				action() {
					router.push({ name: "Container-Index" })
				}
			},
			{
				iconName: "mdi:source-merge",
				iconImage: null,
				key: "pipeline",
				title: "CI/CD 流水线",
				label: "服务",
				tags: ["pipeline", "ci", "cd", "git", "流水线", "构建"],
				action() {
					router.push({ name: "Pipeline-Index" })
				}
			}
		]
	},
	{
		name: "主机管理",
		items: [
			{
				iconName: "mdi:folder-outline",
				iconImage: null,
				key: "Host-Files",
				title: "文件管理",
				label: "主机",
				tags: ["file", "manager", "文件", "目录"],
				action() {
					router.push({ name: "Host-Files" })
				}
			},
			{
				iconName: "mdi:security",
				iconImage: null,
				key: "Host-Firewall",
				title: "防火墙",
				label: "主机",
				tags: ["firewall", "port", "security", "防火墙", "端口", "安全"],
				action() {
					router.push({ name: "Host-Firewall" })
				}
			},
			{
				iconName: "mdi:memory",
				iconImage: null,
				key: "Host-Process",
				title: "进程管理",
				label: "主机",
				tags: ["process", "top", "进程", "任务", "htop"],
				action() {
					router.push({ name: "Host-Process" })
				}
			},
			{
				iconName: "mdi:cogs",
				iconImage: null,
				key: "Toolbox-Daemon",
				title: "守护进程",
				label: "主机",
				tags: ["daemon", "supervisor", "守护进程", "后台"],
				action() {
					router.push({ name: "Toolbox-Daemon" })
				}
			},
			{
				iconName: "mdi:monitor-dashboard",
				iconImage: null,
				key: "Host-Monitor",
				title: "资源监控",
				label: "主机",
				tags: ["monitor", "cpu", "ram", "监控", "资源"],
				action() {
					router.push({ name: "Host-Monitor" })
				}
			}
		]
	},
	{
		name: "本地操作",
		items: [
			{
				iconName: FullScreenIcon,
				iconImage: null,
				key: "action-fullscreen",
				title: "切换全屏",
				label: "操作",
				tags: ["fullscreen", "全屏", "窗口"],
				action() {
					useFullscreenSwitch().toggle()
				}
			},
			{
				iconName: DarkModeIcon,
				iconImage: null,
				key: "action-darkmode",
				title: "切换深色模式",
				label: "操作",
				tags: ["dark", "theme", "深色", "主题", "暗黑"],
				action() {
					useThemeSwitch().toggle()
				}
			}
		]
	}
])

const keywords = computed<string[]>(() => {
	return search.value.length > 1 ? search.value.split(" ").filter(k => k) : []
})

const filteredGroups = computed<Groups>(() => {
	const userMenus = authStore.userMenus || []
	const isSuperAdmin = authStore.role === "SUPER" || authStore.role === "ADMIN" || userMenus.includes("ALL")

	// 判断当前项是否有权限
	const hasPermission = (key: string) => {
		if (isSuperAdmin) return true
		if (key.startsWith("action-")) return true // 本地操作放行
		return userMenus.includes(key)
	}

	// 首先根据权限过滤 items
	const permittedGroups = groups.value
		.map(group => ({
			name: group.name,
			items: group.items.filter(item => hasPermission(item.key))
		}))
		.filter(group => group.items.length > 0)

	if (!keywords.value.length) return permittedGroups

	// 再根据搜索关键词过滤
	return permittedGroups
		.map(group => ({
			name: group.name,
			items: group.items.filter(
				item =>
					keywords.value.some(k => item.title.toLowerCase().includes(k.toLowerCase())) ||
					item.tags?.some(t => keywords.value.some(k => t.toLowerCase().includes(k.toLowerCase())))
			)
		}))
		.filter(group => group.items.length)
})

const filteredFlattenItems = computed<GroupItem[]>(() => {
	return filteredGroups.value.reduce((acc, group) => [...acc, ...group.items], [] as GroupItem[])
})

function openBox(e?: MouseEvent) {
	if (!showSearchBox.value) {
		showSearchBox.value = true

		setTimeout(() => {
			search.value = ""
			activeItem.value = null
		}, 100)
	}
	return e
}

function closeBox() {
	showSearchBox.value = false
	search.value = ""
	activeItem.value = null
}

function callAction(action: () => void) {
	action()
	closeBox()
}

function nextItem() {
	const currentIndex = filteredFlattenItems.value.findIndex(item => item.key === activeItem.value)
	if (currentIndex === filteredFlattenItems.value.length - 1 || activeItem.value === null) {
		activeItem.value = filteredFlattenItems.value[0].key
	} else {
		activeItem.value = filteredFlattenItems.value[currentIndex + 1].key
	}
	centerItem()
}

function prevItem() {
	const currentIndex = filteredFlattenItems.value.findIndex(item => item.key === activeItem.value)
	if (currentIndex === 0 || activeItem.value === null) {
		activeItem.value = filteredFlattenItems.value[filteredFlattenItems.value.length - 1].key
	} else {
		activeItem.value = filteredFlattenItems.value[currentIndex - 1].key
	}
	centerItem()
}

function performAction() {
	const item = filteredFlattenItems.value.find(item => item.key === activeItem.value)
	if (item) {
		callAction(item.action)
	}
}

function centerItem() {
	const element = document.getElementById(activeItem.value?.toString() || "")
	if (element && scrollContent.value) {
		element.scrollIntoView({ block: "nearest" })
	}
}

onMounted(() => {
	const isWindows = getOS() === "Windows"
	commandIcon.value = isWindows ? "CTRL" : "⌘"

	const keys = useMagicKeys()
	const ActiveCMD = isWindows ? keys["ctrl+k"] : keys["cmd+k"]
	const Enter = keys.enter

	useSearchDialog().trigger(openBox)

	whenever(ActiveCMD, () => {
		openBox()
	})

	whenever(Enter, () => {
		if (showSearchBox.value) {
			performAction()
		}
	})
})
const getHighlighterText = (text: string) => {
	if (!search.value) return text

	// 创建全局且忽略大小写的正则
	const regex = new RegExp(`(${search.value})`, "gi")

	// 将匹配到的部分包裹在 span 中
	// 这里使用了 风格的类名，或者直接用你原有的 .highlight
	return text.replace(regex, '<span class="highlight">$1</span>')
}
</script>

<style lang="scss" scoped>
.search-box-modal {
	.search-box {
		border-radius: 4px;

		.search-input {
			height: 50px;
			gap: 20px;
			padding: 20px;

			input {
				background: transparent;
				outline: none;
				border: none;
				min-width: 100px;
			}

			.n-text--code {
				white-space: nowrap;
			}
		}

		.n-divider {
			margin-top: 0;
			margin-bottom: 0;
		}

		.conten-wrap {
			padding-bottom: 30px;

			.group-empty {
				text-align: center;
				padding: 30px 0 40px 0;
			}
			.group {
				padding: 0 10px;
				.group-title {
					opacity: 0.6;
					margin-bottom: 5px;
					padding: 5px 10px;
					padding-top: 20px;
				}
				.group-list {
					.item {
						padding: 7px 10px;
						gap: 10px;
						cursor: pointer;
						border-radius: 10px;
						width: 100%;
						text-align: left;

						.icon {
							width: 28px;
							height: 28px;
							border-radius: 50%;
							background-color: rgba(var(--primary-color-rgb) / 0.15);
							display: flex;
							justify-content: center;
							align-items: center;
						}
						.title {
							font-weight: bold;
						}
						.label {
							opacity: 0.8;
							font-size: 0.9em;
						}

						&.active {
							background-color: var(--hover-color);
						}
						&:hover {
							box-shadow: 0px 0px 0px 1px var(--primary-color) inset;
						}
					}
				}
			}
		}

		.hint-bar {
			font-size: 12px;
			gap: 20px;
			padding: 10px 0;

			.icon {
				background-color: rgba(255, 255, 255, 0.3);
				width: 18px;
				height: 18px;
				padding-top: 1px;
				text-align: center;
				border-radius: 4px;
				display: flex;
				align-items: center;
				justify-content: center;
			}
			.label {
				opacity: 0.7;
			}
		}
	}
}
</style>
