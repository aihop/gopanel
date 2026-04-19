<template>
	<div
		class="box-border flex w-full flex-col overflow-hidden rounded-lg border transition-colors duration-300"
		:style="{ height: height }"
	>
		<div
			v-if="showToolbar"
			class="z-10 flex flex-none items-center justify-between border-b border-[#e0e0e0] bg-[#f7f7f9] px-3 py-2"
		>
			<div class="flex flex-1 items-center gap-4">
				<div class="flex items-center gap-2">
					<span class="whitespace-nowrap text-[12px] font-medium text-[#666]">主题</span>
					<n-select v-model:value="config.theme" size="small" class="w-[180px]" :options="themes" />
				</div>

				<div class="flex items-center gap-2">
					<span class="whitespace-nowrap text-[12px] font-medium text-[#666]">语言</span>
					<n-select
						v-model:value="config.language"
						size="small"
						class="w-[180px]"
						:options="Languages.map(lang => ({ label: lang.label, value: lang.label }))"
						filterable
					/>
				</div>

				<div class="flex items-center gap-2">
					<span class="whitespace-nowrap text-[12px] font-medium text-[#666]">换行</span>
					<n-select
						v-model:value="MONACO_EDITOR_OPTIONS.wordWrap"
						size="small"
						class="w-[180px]"
						:options="wrapOptions"
					/>
				</div>
			</div>

			<div class="flex items-center gap-2">
				<n-button quaternary circle size="small" @click="initEditor">
					<template #icon>
						<div class="i-carbon-code text-[#666]" />
					</template>
				</n-button>
			</div>
		</div>

		<div class="relative min-h-0 w-full flex-1 bg-[#1e1e1e]">
			<vue-monaco-editor
				v-model:value="form.content"
				class="h-full w-full"
				:options="MONACO_EDITOR_OPTIONS"
				:language="config.language"
				:theme="config.theme"
				@mount="handleMount"
			/>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { shallowRef, reactive, ref, watch, computed } from "vue"
import { VueMonacoEditor } from "@guolao/vue-monaco-editor"
import { Languages } from "@/global/mimetype"
import { useI18n } from "vue-i18n"

const props = defineProps({
	modelValue: { type: String, default: "" },
	height: { type: String, default: "55vh" },
	theme: { type: String, default: "vs-dark" },
	language: { type: String, default: "html" },
	showWorker: { type: Boolean, default: true },
	readonly: { type: Boolean, default: false },
	showToolbar: { type: Boolean, default: true }
})

const emit = defineEmits(["update:modelValue", "change"])
const { t } = useI18n()

const form = ref({ content: props.modelValue, path: "" })
const config = reactive({ theme: props.theme, language: props.language })

const MONACO_EDITOR_OPTIONS = reactive({
	automaticLayout: true,
	formatOnType: true,
	formatOnPaste: true,
	wordWrap: "on" as "on" | "off",
	minimap: { enabled: false },
	scrollBeyondLastLine: false,
	padding: { top: 8, bottom: 8 },
	fontSize: 13,
	lineNumbersMinChars: 3,
	readOnly: props.readonly
})

const wrapOptions = computed(() => [
	{ label: t("enable"), value: "on" },
	{ label: t("disable"), value: "off" }
])

const themes = [
	{ label: "Light", value: "vs" },
	{ label: "Dark", value: "vs-dark" }
]

const editorRef = shallowRef()
const handleMount = (editor: any) => {
	editorRef.value = editor
	editor.onDidChangeModelContent(() => {
		const val = editor.getValue()
		form.value.content = val
		emit("update:modelValue", val)
		emit("change", val)
	})
}

const initEditor = () => {
	editorRef.value?.getAction("editor.action.formatDocument").run()
}

watch(
	() => props.modelValue,
	val => {
		if (val !== form.value.content) form.value.content = val
	}
)

watch(
	() => props.readonly,
	val => {
		MONACO_EDITOR_OPTIONS.readOnly = val
		editorRef.value?.updateOptions?.({ readOnly: val })
	}
)

watch(
	() => props.language,
	newLang => {
		if (newLang) config.language = newLang
	}
)

const isJSON = (str: string) => {
	try {
		JSON.parse(str)
		return true
	} catch (e) {
		return false
	}
}

const acceptParams = (params: any) => {
	form.value.content = params.content
	if (params.path) form.value.path = params.path
	if (params.language) config.language = params.language
	if (isJSON(params.content)) config.language = "json"
}

const scrollToBottom = () => {
	const editor = editorRef.value
	if (!editor) return
	const model = editor.getModel?.()
	const lineCount = model?.getLineCount?.() || 1
	editor.revealLine?.(lineCount)
	editor.setPosition?.({ lineNumber: lineCount, column: model?.getLineMaxColumn?.(lineCount) || 1 })
}

defineExpose({ acceptParams, initEditor, scrollToBottom, editorRef })
</script>

<style scoped>
:deep(.vue-monaco-editor) {
	width: 100% !important;
	height: 100% !important;
}
</style>
