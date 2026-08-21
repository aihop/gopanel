#!/usr/bin/env node
/**
 * i18n-check.mjs
 *
 * 扫 admin/src/views 与 admin/src/components 下的 .vue / .ts，
 * 找出仍存在的硬编码中文字符串（CJK Unified Ideographs U+4E00-U+9FFF）。
 * 文件命中后按 file:line 打印并以 exit 1 失败，便于在 CI 上 fail build。
 *
 * 用法：node scripts/i18n-check.mjs [--baseline <file>]
 *
 *   --baseline <file>  与已记录的存量列表对比：
 *                      新增条目 → 失败（不允许继续新增硬编码）
 *                      列表中存在但当前未命中 → 通过
 *                      用于把 A3 落地时刻的存量固化为基线，
 *                      后续各模块迁移后只需刷新基线即可。
 */
import fs from "node:fs"
import path from "node:path"
import process from "node:process"
import url from "node:url"

const __dirname = path.dirname(url.fileURLToPath(import.meta.url))
const ADMIN_ROOT = path.resolve(__dirname, "..")
const SCAN_DIRS = ["src/views", "src/components"]
const EXCLUDE_DIRS = new Set(["node_modules", ".output", "dist", ".git"])
const EXCLUDE_FILE_PATTERNS = [
	// 测试文件允许出现中文字面量作为 fixture
	/\.spec\.[jt]sx?$/i,
	/_test\.[jt]sx?$/i,
	/\.test\.[jt]sx?$/i,
]
// i18n 源文件本身就是翻译 key 的来源，自身允许包含中文。
const ALWAYS_SKIP_SUBSTRINGS = [
	path.join("i18n", "locales"),
	path.join("i18n", "locales" + path.sep),
]
// CJK Unified Ideographs Basic block + Extension A 起点
const CJK_REGEX = /[\u3400-\u4DBF\u4E00-\u9FFF]/g

/**
 * 递归遍历 dir，收集 .vue / .ts 文件。
 */
function* walk(dir) {
	if (!fs.existsSync(dir)) return
	const stack = [dir]
	while (stack.length) {
		const cur = stack.pop()
		let entries
		try {
			entries = fs.readdirSync(cur, { withFileTypes: true })
		} catch {
			continue
		}
		for (const entry of entries) {
			if (EXCLUDE_DIRS.has(entry.name)) continue
			const full = path.join(cur, entry.name)
			if (entry.isDirectory()) {
				stack.push(full)
			} else if (/\.(vue|ts)$/i.test(entry.name)) {
				yield full
			}
		}
	}
}

function shouldSkip(file) {
	const relative = path.relative(ADMIN_ROOT, file)
	if (EXCLUDE_FILE_PATTERNS.some(re => re.test(file))) return true
	if (ALWAYS_SKIP_SUBSTRINGS.some(sub => relative.includes(sub))) return true
	return false
}

function isCommentLine(line) {
	const trimmed = line.trim()
	return (
		trimmed.startsWith("//") ||
		trimmed.startsWith("/*") ||
		trimmed.startsWith("*") ||
		trimmed.startsWith("<!--")
	)
}

// stripCommentTail 把 // 之后到行尾的内容去掉（保留前导字符串/模板文本）。
// 不处理块注释：多行 /* ... */ 通常跨行，跨行单测留给 isCommentLine。
function stripCommentTail(line) {
	const idx = line.indexOf("//")
	if (idx < 0) return line
	// 字符串字面量内的 // 不应被截断。
	const before = line.slice(0, idx)
	const dq = (before.match(/"/g) || []).length
	const sq = (before.match(/'/g) || []).length
	const bt = (before.match(/`/g) || []).length
	// 偶数次引号 → 当前位置不在字符串内
	const inString =
		(dq % 2 === 1 && sq % 2 === 0 && bt % 2 === 0) ||
		(sq % 2 === 1 && dq % 2 === 0 && bt % 2 === 0) ||
		(bt % 2 === 1 && dq % 2 === 0 && sq % 2 === 0)
	if (inString) return line
	return before
}

function collectHits(file) {
	const text = fs.readFileSync(file, "utf8")
	const lines = text.split(/\r?\n/)
	const hits = []
	lines.forEach((line, i) => {
		if (isCommentLine(line)) return
		const stripped = stripCommentTail(line)
		const cjkMatches = stripped.match(CJK_REGEX)
		if (!cjkMatches || cjkMatches.length === 0) return
		const trimmed = stripped.trim()
		hits.push({
			file: path.relative(ADMIN_ROOT, file),
			line: i + 1,
			text: trimmed,
			chars: cjkMatches.length,
		})
	})
	return hits
}

function parseArgs(argv) {
	const opts = { baseline: null, emitBaseline: null }
	for (let i = 2; i < argv.length; i++) {
		const arg = argv[i]
		if (arg === "--baseline" || arg === "-b") {
			opts.baseline = argv[++i]
		} else if (arg === "--emit-baseline" || arg === "-e") {
			// 默认相对 cwd，便于从仓库根运行；传入绝对路径则原样使用
			opts.emitBaseline = argv[++i] || "admin/scripts/i18n-baseline.json"
		} else if (arg === "--help" || arg === "-h") {
			console.log(
				"Usage: node scripts/i18n-check.mjs [--baseline <file>] [--emit-baseline <file>]\n\n" +
					"Scans admin/src for hardcoded Chinese strings; exits 1 if any are found.\n" +
					"--baseline      fail only on NEW occurrences vs an existing JSON baseline.\n" +
					"--emit-baseline write the current list of occurrences to a JSON file.",
			)
			process.exit(0)
		}
	}
	return opts
}

function loadBaseline(file) {
	if (!file) return null
	const abs = path.isAbsolute(file) ? file : path.resolve(process.cwd(), file)
	if (!fs.existsSync(abs)) {
		throw new Error(
			`baseline file not found: ${abs}\n` +
				`  Pass --emit-baseline <file> first to generate it.`,
		)
	}
	const raw = fs.readFileSync(abs, "utf8")
	try {
		return JSON.parse(raw)
	} catch {
		throw new Error(`baseline file is not valid JSON: ${abs}`)
	}
}

function main() {
	const opts = parseArgs(process.argv)
	const allHits = []
	for (const sub of SCAN_DIRS) {
		const root = path.join(ADMIN_ROOT, sub)
		for (const file of walk(root)) {
			if (shouldSkip(file)) continue
			const hits = collectHits(file)
			allHits.push(...hits)
		}
	}

	if (opts.emitBaseline) {
		const out = path.isAbsolute(opts.emitBaseline)
			? opts.emitBaseline
			: path.resolve(process.cwd(), opts.emitBaseline)
		const payload = JSON.stringify(allHits, null, 2)
		fs.writeFileSync(out, payload + "\n")
		console.log(
			`✓ i18n baseline written: ${path.relative(process.cwd(), out)} (${allHits.length} entries)`,
		)
		return
	}

	// 按文件聚合，方便阅读
	const byFile = new Map()
	for (const hit of allHits) {
		if (!byFile.has(hit.file)) byFile.set(hit.file, [])
		byFile.get(hit.file).push(hit)
	}

	const baseline = loadBaseline(opts.baseline)
	if (baseline) {
		// 基线模式：基线已存在的允许保留；基线外的（新增硬编码）必须 fail
		const known = new Set(
			baseline.map(entry => `${entry.file}:${entry.line}`),
		)
		const fresh = allHits.filter(h => !known.has(`${h.file}:${h.line}`))
		if (fresh.length > 0) {
			console.error(
				`✖ i18n-check failed: ${fresh.length} new hardcoded Chinese line(s) outside baseline.`,
			)
			for (const hit of fresh) {
				console.error(`  ${hit.file}:${hit.line}: ${hit.text.slice(0, 120)}`)
			}
			console.error(
				"\nHint: replace the string with t('your.key') and add the entry to admin/src/i18n/locales/*.",
			)
			process.exit(1)
		}
		console.log(
			`✓ i18n-check: ${allHits.length} known hardcoded Chinese line(s) match baseline; no new occurrences.`,
		)
		return
	}

	if (allHits.length === 0) {
		console.log("✓ i18n-check: no hardcoded Chinese strings found.")
		return
	}

	console.error(
		`✖ i18n-check failed: ${allHits.length} hardcoded Chinese line(s) across ${byFile.size} file(s).`,
	)
	const sortedFiles = Array.from(byFile.keys()).sort()
	for (const file of sortedFiles) {
		const hits = byFile.get(file)
		console.error(`\n${file}`)
		for (const hit of hits) {
			console.error(`  L${hit.line}  ${hit.text.slice(0, 120)}`)
		}
	}
	console.error(
		"\nHint: replace strings with t('your.key') and add the entry to admin/src/i18n/locales/*.",
	)
	console.error(
		"      To bootstrap a baseline for the existing 190+ files, run:\n" +
			"        node scripts/i18n-check.mjs --baseline admin/scripts/i18n-baseline.json",
	)
	process.exit(1)
}

main()