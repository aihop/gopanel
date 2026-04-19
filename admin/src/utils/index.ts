import type { Component } from "vue"
import process from "node:process"
import Icon from "@/components/common/Icon.vue"
import { isMobile as detectMobile } from "detect-touch-device"
import { h } from "vue"

export type OS = "Unknown" | "Windows" | "MacOS" | "UNIX" | "Linux"

// Transform File Instance in base64 string
export function file2Base64(blob: Blob): Promise<string> {
	return new Promise((resolve, reject) => {
		const reader = new FileReader()
		reader.readAsDataURL(blob)
		reader.onload = () => resolve(reader.result as string)
		reader.onerror = error => reject(error)
	})
}

export function isEnvDev() {
	return process.env.NODE_ENV === "development"
}
export function isEnvTest() {
	return process.env.NODE_ENV === "test"
}
export function isEnvProd() {
	return process.env.NODE_ENV === "production"
}

export function isMobile() {
	return detectMobile
}

export function renderIcon(icon: Component | string) {
	if (typeof icon === "string") {
		return () => h(Icon, { name: icon })
	} else {
		return () => h(Icon, null, { default: () => h(icon) })
	}
}

export function getOS(): OS {
	let os: OS = "Unknown"
	if (navigator.userAgent.includes("Win")) os = "Windows"
	if (navigator.userAgent.includes("Mac")) os = "MacOS"
	if (navigator.userAgent.includes("X11")) os = "UNIX"
	if (navigator.userAgent.includes("Linux")) os = "Linux"

	return os
}

export function delay(t: number) {
	return new Promise(res => setTimeout(res, t))
}
