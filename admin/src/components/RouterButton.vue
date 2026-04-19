<template>
  <div class="mb-6 rounded-[28px]">
    <div class="flex w-full flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
      <div class="flex flex-1 flex-wrap gap-3">
        <label
          v-for="(button, index) in buttonArray"
          :key="index"
          class="cursor-pointer bg-base-accent border-base-accent min-w-[120px] w-full sm:w-auto rounded-[20px]  p-1 transition-all duration-200 ease-out hover:-translate-y-[1px] hover:border-blue-300/90 hover:shadow-[0_14px_30px_rgba(59,130,246,0.12)]"
          :class="[
						activeName === button.label
							? 'border-blue-500/30 bg-gradient-to-br from-blue-100/95 to-blue-50/92 shadow-[0_16px_34px_rgba(37,99,235,0.14)]'
							: 'bg-gradient-to-b from-slate-50/98 to-slate-100/92 dark:bg-base-100'
					]"
        >
          <input
            v-model="activeName"
            type="radio"
            class="hidden"
            :name="name"
            :value="button.label"
            @change="handleChange(button.label)"
          />
          <div
            class="flex min-h-[54px] sm:min-h-[58px] items-center justify-between gap-3 rounded-2xl px-[18px] text-[15px] font-semibold leading-tight"
            :class="activeName === button.label ? 'text-blue-600' : 'text-slate-600'"
          >
            <div class="whitespace-nowrap">{{ button.label }}</div>
            <n-badge
              v-if="button.count"
              :value="button.count"
              :max="999"
            />
          </div>
        </label>
      </div>
      <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
        <slot name="route-button" />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue"
import { useRouter } from "vue-router"

defineOptions({ name: "RouterButton" })
const props = defineProps<{
	buttons: RouterButton[]
}>()

const emit = defineEmits(["update:active"])

const name = `RouterButton${new Date().getTime()}`

interface RouterButton {
	label: string
	path?: string
	name?: string
	count?: number
}

const buttonArray = computed(() => props.buttons)
const router = useRouter()
const activeName = ref("")

function handleChange(label: string) {
	const btn = buttonArray.value.find(b => b.label === label)
	if (!btn) return
	if (btn.path) router.push({ path: btn.path })
	else if (btn.name) router.push({ name: btn.name })
	activeName.value = label
	emit("update:active", label)
}

onMounted(() => {
	const arr = buttonArray.value
	if (!arr.length) return
	const current = router.currentRoute.value.path
	const match = arr.find(b => b.path && current.startsWith(b.path))
	activeName.value = match?.label || arr[0].label
})
</script>

<style lang="scss" scoped>
.router_card {
	:deep(.n-card__content) {
		padding: 22px 24px;
	}
}

:deep(.n-badge .n-badge-sup) {
	box-shadow: none;
}
</style>
