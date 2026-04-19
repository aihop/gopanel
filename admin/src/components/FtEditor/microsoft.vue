<template>
	<n-form inline :model="config" v-if="showWorker" class="flex overflow-auto" style="flex-wrap: nowrap">
		<n-form-item :label="$t('config.theme')">
			<n-select
				v-model:value="config.theme"
				style="width: 140px"
				:options="themes"
				label-field="label"
				value-field="value"
				@update:value="changeTheme"
			/>
		</n-form-item>
		<n-form-item :label="$t('language')">
			<n-select
				v-model:value="config.language"
				style="width: 120px"
				:options="Languages.map(lang => ({ label: lang.label, value: lang.label }))"
				label-field="label"
				value-field="value"
				@update:value="changeLanguage"
			/>
		</n-form-item>
		<n-form-item :label="$t('file.eol')">
			<n-select
				v-model:value="config.eol"
				style="width: 120px"
				:options="eols"
				label-field="label"
				value-field="value"
				@update:value="changeEOL"
			/>
		</n-form-item>
		<n-form-item :label="$t('file.wordWrap')">
			<n-select
				v-model:value="config.wordWrap"
				style="width: 100px"
				@update:value="changeWarp"
				:options="options"
			></n-select>
		</n-form-item>
	</n-form>
	<div id="codeBox" :style="{ height }"></div>
</template>

<script lang="ts" setup>
import * as monaco from "monaco-editor"
import { nextTick, onBeforeUnmount, reactive, ref } from "vue"
import { Languages } from "@/global/mimetype"
import jsonWorker from "monaco-editor/esm/vs/language/json/json.worker?worker"
import cssWorker from "monaco-editor/esm/vs/language/css/css.worker?worker"
import htmlWorker from "monaco-editor/esm/vs/language/html/html.worker?worker"
import tsWorker from "monaco-editor/esm/vs/language/typescript/ts.worker?worker"
import EditorWorker from "monaco-editor/esm/vs/editor/editor.worker?worker"
import { useI18n } from "vue-i18n"

const { t } = useI18n()

const options = [
	{
		label: t("enable"),
		value: "on"
	},
	{
		label: t("disable"),
		value: "off"
	}
]

let editor: any | undefined
const emit = defineEmits(["close"])
;(window as any).MonacoEnvironment = {
	getWorker(workerId, label) {
		if (label === "json") {
			return new jsonWorker()
		}
		if (label === "css" || label === "scss" || label === "less") {
			return new cssWorker()
		}
		if (label === "html" || label === "handlebars" || label === "razor") {
			return new htmlWorker()
		}
		if (["typescript", "javascript"].includes(label)) {
			return new tsWorker()
		}
		return new EditorWorker()
	}
}

interface EditProps {
	language: string
	content: string
	path: string
	name: string
}

interface EditorConfig {
	theme: string
	language: string
	eol: number
	wordWrap: WordWrapOptions
}

const props: any = defineProps({
	quickSave: Function,
	height: {
		type: String,
		default: "55vh"
	},
	theme: {
		type: String,
		default: "vs-dark"
	},
	language: {
		type: String,
		default: "html"
	},
	showWorker: {
		type: Boolean,
		default: true
	}
})

const fileName = ref("")

type WordWrapOptions = "off" | "on" | "wordWrapColumn" | "bounded"

const config = reactive<EditorConfig>({
	theme: props.theme,
	language: props.language,
	eol: monaco.editor.EndOfLineSequence.LF,
	wordWrap: "on"
})

const eols = [
	{
		label: "LF (Linux)",
		value: monaco.editor.EndOfLineSequence.LF
	},
	{
		label: "CRLF (Windows)",
		value: monaco.editor.EndOfLineSequence.CRLF
	}
]

const themes = [
	{
		label: "Light",
		value: "vs"
	},
	{
		label: "Dark",
		value: "vs-dark"
	},
	{
		label: "High Dark",
		value: "hc-black"
	}
]

let form = ref({
	content: "",
	path: ""
})

const changeLanguage = () => {
	if (editor) {
		monaco.editor.setModelLanguage(editor.getModel() as any, config.language)
	}
}

const changeTheme = () => {
	monaco.editor.setTheme(config.theme)
}

const changeEOL = () => {
	if (editor) {
		editor.getModel()?.pushEOL(config.eol)
	}
}

const changeWarp = () => {
	if (editor) {
		editor.updateOptions({
			wordWrap: config.wordWrap
		})
	}
}
const initEditor = () => {
	if (editor) {
		editor.dispose()
	}
	nextTick(() => {
		const codeBox = document.getElementById("codeBox")
		editor = monaco.editor.create(codeBox as HTMLElement, {
			theme: config.theme,
			value: form.value.content,
			readOnly: false,
			automaticLayout: true,
			language: config.language,
			folding: true,
			roundedSelection: false,
			overviewRulerBorder: false,
			wordWrap: "on"
		})
		editor.onDidChangeModelContent(() => {
			if (editor) {
				form.value.content = editor.getValue()
			}
		})
		editor.getModel()?.pushEOL(config.eol)

		editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, props.quickSave)
	})
}

const acceptParams = (props: EditProps) => {
	form.value.content = props.content
	if (props.path) form.value.path = props.path
	if (props.language) config.language = props.language
	if (props.name) fileName.value = props.name
	config.eol = monaco.editor.EndOfLineSequence.LF
}

onBeforeUnmount(() => {
	if (editor) {
		editor.dispose()
	}
})
const getForm = () => form.value

defineExpose({ acceptParams, initEditor, getForm })
</script>
