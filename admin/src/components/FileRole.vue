<template>
	<div>
		<n-form ref="ruleForm" :model="form" label-placement="left" label-width="100px" :rules="rules">
			<n-form-item :label="$t('file.owner')">
				<n-space>
					<n-checkbox v-model:checked="form.owner.r" :label="$t('file.rRole')" />
					<n-checkbox v-model:checked="form.owner.w" :label="$t('file.wRole')" />
					<n-checkbox v-model:checked="form.owner.x" :label="$t('file.xRole')" />
				</n-space>
			</n-form-item>

			<n-form-item :label="$t('file.group')">
				<n-space>
					<n-checkbox v-model:checked="form.group.r" :label="$t('file.rRole')" />
					<n-checkbox v-model:checked="form.group.w" :label="$t('file.wRole')" />
					<n-checkbox v-model:checked="form.group.x" :label="$t('file.xRole')" />
				</n-space>
			</n-form-item>

			<n-form-item :label="$t('file.public')">
				<n-space>
					<n-checkbox v-model:checked="form.public.r" :label="$t('file.rRole')" />
					<n-checkbox v-model:checked="form.public.w" :label="$t('file.wRole')" />
					<n-checkbox v-model:checked="form.public.x" :label="$t('file.xRole')" />
				</n-space>
			</n-form-item>

			<n-form-item :label="$t('file.role')" path="mode">
				<n-input v-model:value="form.mode" maxlength="4" @input="changeMode" />
			</n-form-item>
		</n-form>
	</div>
</template>

<script setup lang="ts">
import { computed, ref, toRefs, watch, onMounted, onUpdated, reactive } from "vue"
import { useI18n } from "vue-i18n"

interface Role {
	r: boolean
	w: boolean
	x: boolean
}
interface RoleForm {
	owner: Role
	group: Role
	public: Role
	mode: string
}
interface Props {
	mode: string
}

const { t } = useI18n()

const props = withDefaults(defineProps<Props>(), {
	mode: "0755"
})

const rules = reactive({
	mode: [
		{
			validator: (_rule: any, value: string) => {
				const reg = /^[0-7]{4}$/
				if (value && reg.test(value)) return Promise.resolve()
				return Promise.reject(new Error(t("commons.msg.invalid") || "invalid mode"))
			},
			trigger: ["blur", "input"]
		}
	]
})

const roles = ref<string[]>(["0", "1", "2", "3", "4", "5", "6", "7"])

const { mode } = toRefs(props)
const ruleForm = ref<any | null>(null)

const form = ref<RoleForm>({
	owner: { r: true, w: true, x: true },
	group: { r: true, w: true, x: true },
	public: { r: true, w: false, x: true },
	mode: "0755"
})

const em = defineEmits(["getMode"])

const calculate = (role: Role) => {
	let num = 0
	if (role.r) num += 4
	if (role.w) num += 2
	if (role.x) num += 1
	return num
}

const getRole = computed(() => {
	return (
		"0" +
		String(calculate(form.value.owner)) +
		String(calculate(form.value.group)) +
		String(calculate(form.value.public))
	)
})

watch(
	() => getRole.value,
	newVal => {
		form.value.mode = newVal
	}
)

watch(
	() => form.value.mode,
	newVal => {
		em("getMode", Number.parseInt(newVal, 8))
	}
)

const getRoleNum = (roleStr: string, role: Role) => {
	if (roles.value.indexOf(roleStr) < 0) return
	switch (roleStr) {
		case "0":
			role.x = false
			role.w = false
			role.r = false
			break
		case "1":
			role.x = true
			role.w = false
			role.r = false
			break
		case "2":
			role.x = false
			role.w = true
			role.r = false
			break
		case "3":
			role.x = true
			role.w = true
			role.r = false
			break
		case "4":
			role.x = false
			role.w = false
			role.r = true
			break
		case "5":
			role.x = true
			role.w = false
			role.r = true
			break
		case "6":
			role.x = false
			role.w = true
			role.r = true
			break
		case "7":
			role.x = true
			role.w = true
			role.r = true
			break
	}
}

const changeMode = (val: string) => {
	if (!val || val.length !== 4) return
	getRoleNum(val[1], form.value.owner)
	getRoleNum(val[2], form.value.group)
	getRoleNum(val[3], form.value.public)
}

const updateMode = () => {
	form.value.mode = mode.value
	changeMode(form.value.mode)
}

onMounted(() => {
	updateMode()
})

onUpdated(() => {
	updateMode()
})
</script>
