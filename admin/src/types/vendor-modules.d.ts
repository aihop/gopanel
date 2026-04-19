declare module "@floating-ui/dom" {
	export const offset: (value: number) => any
}

declare module "shepherd.js" {
	const Shepherd: any
	export default Shepherd
}

declare module "shepherd.js/dist/css/shepherd.css"

declare module "monaco-editor" {
	const monaco: any
	export = monaco
}

declare module "monaco-editor/esm/vs/editor/editor.api" {
	const monaco: any
	export = monaco
}

declare module "monaco-editor/esm/vs/*" {
	const value: any
	export default value
}

declare module "monaco-editor/esm/vs/language/json/json.worker?worker" {
	const WorkerFactory: any
	export default WorkerFactory
}

declare module "monaco-editor/esm/vs/language/css/css.worker?worker" {
	const WorkerFactory: any
	export default WorkerFactory
}

declare module "monaco-editor/esm/vs/language/html/html.worker?worker" {
	const WorkerFactory: any
	export default WorkerFactory
}

declare module "monaco-editor/esm/vs/language/typescript/ts.worker?worker" {
	const WorkerFactory: any
	export default WorkerFactory
}

declare module "monaco-editor/esm/vs/editor/editor.worker?worker" {
	const WorkerFactory: any
	export default WorkerFactory
}

declare global {
	namespace monaco {
		namespace editor {
			type IStandaloneCodeEditor = any
			type ITextModel = any
			const EndOfLineSequence: any
		}
		const KeyMod: any
		const KeyCode: any
	}

	interface Window {
		MonacoEnvironment?: any
	}
}
