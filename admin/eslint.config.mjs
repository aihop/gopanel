import js from "@eslint/js"
import tsParser from "@typescript-eslint/parser"
import vue from "eslint-plugin-vue"
import globals from "globals"

const sharedLanguageOptions = {
	ecmaVersion: "latest",
	sourceType: "module",
	globals: {
		...globals.browser,
		...globals.node,
	},
}

export default [
	{
		ignores: ["dist/**", "coverage/**", "node_modules/**"],
	},
	{
		...js.configs.recommended,
		languageOptions: sharedLanguageOptions,
	},
	...vue.configs["flat/recommended"],
	{
		files: ["**/*.ts", "**/*.tsx"],
		languageOptions: {
			...sharedLanguageOptions,
			parser: tsParser,
		},
		rules: {
			"no-undef": "off",
			"no-unused-vars": "off",
		},
	},
	{
		files: ["**/*.vue"],
		languageOptions: {
			...sharedLanguageOptions,
			parserOptions: {
				parser: tsParser,
			},
		},
		rules: {
			"no-undef": "off",
			"no-unused-vars": "off",
			"vue/require-default-prop": "off",
		},
	},
	{
		rules: {
			"comma-dangle": ["error", "always-multiline"],
		},
	},
]
