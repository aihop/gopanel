import type { OutputAsset, OutputBundle, OutputChunk, Plugin } from "rollup"
import process from "node:process"
import { fileURLToPath, URL } from "node:url"
import vue from "@vitejs/plugin-vue"
import vueJsx from "@vitejs/plugin-vue-jsx"
import { VueHooksPlusResolver } from "@vue-hooks-plus/resolvers"
import AutoImport from "unplugin-auto-import/vite"
import { NaiveUiResolver } from "unplugin-vue-components/resolvers"
import Components from "unplugin-vue-components/vite"
import { defineConfig, loadEnv } from "vite"
import VueDevTools from "vite-plugin-vue-devtools"
import svgLoader from "vite-svg-loader"
// import { visualizer } from "rollup-plugin-visualizer"
// import viteCompression from "vite-plugin-compression"

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
	// Load env file based on `mode` in the current working directory.
	// Set the third parameter to '' to load all env regardless of the `VITE_` prefix.
	process.env = { ...process.env, ...loadEnv(mode, process.cwd(), "") }

	return {
		base: process.env.NODE_ENV === "production" ? "/" : "/",
		server: process.env.NODE_ENV === 'production'
? {}
: {
			port: 5176,
			proxy: {
				"/api": {
					target: "http://127.0.0.1:5470/api",
					changeOrigin: true,
					ws: true,
					rewrite: (path: string) => path.replace(/^\/api/, "")
				}
			},
			host: "0.0.0.0"
		},
		plugins: [
			vue({
				script: {
					defineModel: true
				}
			}),
			vueJsx(),
			VueDevTools(),
			svgLoader(),
			Components({
				dirs: ["src/components/common"],
				dts: "src/unplugin.components.d.ts",
				resolvers: [NaiveUiResolver()]
			}),
			AutoImport({
				imports: [
					"vue",
					"vue-router",
					"@vueuse/core",
					"@vueuse/core",
					"pinia",
					"vue-i18n",
					{
						"naive-ui": ["useDialog", "useMessage", "useNotification", "useLoadingBar"]
					},
					{
						"@/enums": ["getEnumDesc"]
					}
				],
				dts: "src/auto-imports.d.ts",
				dirs: ["src/composables", "src/store", "src/api"],
				resolvers: [VueHooksPlusResolver()],
				vueTemplate: true
			}),
			// 打包体积分析插件
				// visualizer({
				// 	open: false, // build 后不自动打开浏览器
				// 	gzipSize: true, // 分析 gzip 大小
				// 	brotliSize: true, // 分析 brotli 大小
				// 	filename: "dist/stats.html" // 生成的分析报告文件路径
				// }),
				// gzip 压缩
				// viteCompression({
				// 	verbose: true,
				// 	disable: false,
				// 	threshold: 10240,  
				// 	algorithm: 'gzip',
				// 	ext: '.gz',
				// }),
				// brotli 压缩（拥有更高的压缩率）
				// viteCompression({
				// 	verbose: true,
				// 	disable: false,
				// 	threshold: 10240,
				// 	algorithm: 'brotliCompress',
				// 	ext: '.br',
				// })
			],
		resolve: {
			alias: [
				{ find: "@", replacement: fileURLToPath(new URL("./src", import.meta.url)) },
			]
		},
		optimizeDeps: {
			include: ["fast-deep-equal"]
		},
		define: {
			__APP_ENV__: JSON.stringify(process.env.APP_ENV),
			__APP_BRAND__: JSON.stringify(process.env.VITE_APP_BRAND || "GoPanel")
		},
		css: {
			preprocessorOptions: {
				scss: {
					silenceDeprecations: ["legacy-js-api", "import"],
					api: "modern-compiler"
				}
			}
		},
		build: {
			rollupOptions: {
				output: {
					manualChunks: (id) => {
						// 解决 monaco-editor 极度臃肿的问题：按 worker 和 editor 拆分
						if (id.includes('node_modules/monaco-editor')) {
							if (id.includes('basic-languages')) {
								return 'monaco-languages';
							}
							if (id.includes('editor/contrib')) {
								return 'monaco-contrib';
							}
							return 'monaco-core';
						}
						if (id.includes('node_modules/@guolao')) {
							return 'editor-vue';
						}
						// 显式抽离 naive-ui 的 DataTable，避免其与主业务块混在一起
						if (id.includes('node_modules/naive-ui/es/data-table') || id.includes('node_modules/treemate')) {
							return 'naive-ui-data-table';
						}
						// 其它通用工具库
						if (id.includes('node_modules/lodash') || id.includes('node_modules/dayjs') || id.includes('node_modules/colord') || id.includes('node_modules/crypto-js') || id.includes('node_modules/validator')) {
							return 'util-vendor';
						}
					}
				}
			},
			chunkSizeWarningLimit: 2000 // 将警告阈值提高到2MB
		}
	}
})
